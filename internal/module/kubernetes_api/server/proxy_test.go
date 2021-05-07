package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/kubernetes_api/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/prototool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_kubernetes_api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	jobToken        = "asdfgasdfxadf"
	requestPath     = "/api/bla"
	requestPayload  = "asdfndaskjfadsbfjsadhvfjhavfjasvf"
	responsePayload = "jknkjnjkasdnfkjasdnfkasdnfjnkjn"
	queryParamValue = "query-param-value with a space"
	queryParamName  = "q with a space"
)

func TestProxy_JobTokenErrors(t *testing.T) {
	tests := []struct {
		name string
		auth []string
	}{
		{
			name: "missing header",
		},
		{
			name: "multiple headers",
			auth: []string{"a", "b"},
		},
		{
			name: "invalid format",
			auth: []string{"Token asdfadsf"},
		},
		{
			name: "invalid format",
			auth: []string{"Bearer asdfadsf"},
		},
		{
			name: "invalid agent id",
			auth: []string{"Bearer ci:asdf:as"},
		},
		{
			name: "empty token",
			auth: []string{"Bearer ci:1:"},
		},
		{
			name: "unknown token type",
			auth: []string{"Bearer blabla:1:asd"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, client, req, _ := setupProxyWithHandler(t, "", func(w http.ResponseWriter, r *http.Request) {
				t.Fail() // unexpected invocation
			})
			req.Header.Del("Authorization")
			if len(tc.auth) > 0 { // nolint: scopelint
				req.Header["Authorization"] = tc.auth // nolint: scopelint
			}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.EqualValues(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}

func TestProxy_UnknownJobToken(t *testing.T) {
	_, _, client, req, _ := setupProxyWithHandler(t, "", func(w http.ResponseWriter, r *http.Request) {
		assertToken(t, r)
		w.WriteHeader(http.StatusUnauthorized) // pretend the token is invalid
	})
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.EqualValues(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProxy_ForbiddenJobToken(t *testing.T) {
	_, _, client, req, _ := setupProxyWithHandler(t, "", func(w http.ResponseWriter, r *http.Request) {
		assertToken(t, r)
		w.WriteHeader(http.StatusForbidden) // pretend the token is forbidden
	})
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.EqualValues(t, http.StatusForbidden, resp.StatusCode)
}

func TestProxy_ServerError(t *testing.T) {
	api, _, client, req, _ := setupProxyWithHandler(t, "", func(w http.ResponseWriter, r *http.Request) {
		assertToken(t, r)
		w.WriteHeader(http.StatusBadGateway) // pretend there is some weird error
	})
	api.EXPECT().
		HandleProcessingError(gomock.Any(), gomock.Any(), gomock.Any(), matcher.ErrorEq("error kind: 0; status: 502"))
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.EqualValues(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestProxy_NoExpectedUrlPathPrefix(t *testing.T) {
	_, _, client, req, _ := setupProxyWithHandler(t, "/bla", defaultGitLabHandler(t))
	req.URL.Path = requestPath
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
}

func TestProxy_ForbiddenAgentId(t *testing.T) {
	_, _, client, req, _ := setupProxy(t)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s:%d:%s", tokenTypeCi, 15 /* disallowed id */, jobToken))
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.EqualValues(t, http.StatusForbidden, resp.StatusCode)
}

func TestProxy_HappyPathWithoutUrlPrefix(t *testing.T) {
	testProxyHappyPath(t, "")
}

func TestProxy_HappyPathWithUrlPrefix(t *testing.T) {
	testProxyHappyPath(t, "/bla")
}

func testProxyHappyPath(t *testing.T, urlPathPrefix string) {
	_, k8sClient, client, req, requestCount := setupProxyWithHandler(t, urlPathPrefix, defaultGitLabHandler(t))
	requestCount.EXPECT().Inc()
	mrClient := mock_kubernetes_api.NewMockKubernetesApi_MakeRequestClient(gomock.NewController(t))
	mrCall := k8sClient.EXPECT().
		MakeRequest(gomock.Any()).
		DoAndReturn(func(ctx context.Context, opts ...grpc.CallOption) (rpc.KubernetesApi_MakeRequestClient, error) {
			requireCorrectOutgoingMeta(t, ctx)
			return mrClient, nil
		})
	gomock.InOrder(append([]*gomock.Call{mrCall}, mockSendStream(t, mrClient,
		&grpctool.HttpRequest{
			Message: &grpctool.HttpRequest_Header_{
				Header: &grpctool.HttpRequest_Header{
					Request: &prototool.HttpRequest{
						Method: http.MethodPost,
						Header: map[string]*prototool.Values{
							"Req-Header": {
								Value: []string{"x1", "x2"},
							},
							"Accept-Encoding": { // added by the Go client
								Value: []string{"gzip"},
							},
							"User-Agent": {
								Value: []string{"test-agent"},
							},
							"Content-Length": { // added by the Go client
								Value: []string{strconv.Itoa(len(requestPayload))},
							},
						},
						UrlPath: requestPath,
						Query: map[string]*prototool.Values{
							queryParamName: {
								Value: []string{queryParamValue},
							},
						},
					},
				},
			},
		},
		&grpctool.HttpRequest{
			Message: &grpctool.HttpRequest_Data_{
				Data: &grpctool.HttpRequest_Data{
					Data: []byte(requestPayload),
				},
			},
		},
		&grpctool.HttpRequest{
			Message: &grpctool.HttpRequest_Trailer_{
				Trailer: &grpctool.HttpRequest_Trailer{},
			},
		},
	)...)...)
	gomock.InOrder(append([]*gomock.Call{mrCall}, mockRecvStream(mrClient,
		&grpctool.HttpResponse{
			Message: &grpctool.HttpResponse_Header_{
				Header: &grpctool.HttpResponse_Header{
					Response: &prototool.HttpResponse{
						StatusCode: http.StatusOK,
						Status:     "ok",
						Header: map[string]*prototool.Values{
							"Resp-Header": {
								Value: []string{"a1", "a2"},
							},
							"Content-Type": {
								Value: []string{"application/octet-stream"},
							},
							"Date": {
								Value: []string{"NOW!"},
							},
						},
					},
				},
			},
		},
		&grpctool.HttpResponse{
			Message: &grpctool.HttpResponse_Data_{
				Data: &grpctool.HttpResponse_Data{
					Data: []byte(responsePayload),
				},
			},
		},
		&grpctool.HttpResponse{
			Message: &grpctool.HttpResponse_Trailer_{
				Trailer: &grpctool.HttpResponse_Trailer{},
			},
		},
	)...)...)

	req.Header.Set("Req-Header", "x1")
	req.Header.Add("Req-Header", "x2")
	req.Header.Set("User-Agent", "test-agent") // added manually to override what is added by the Go client
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, responsePayload, string(respData))
	resp.Header.Del("Date")
	assert.Empty(t, cmp.Diff(map[string][]string{
		"Resp-Header":    {"a1", "a2"},
		"Content-Length": {"31"},
		"Content-Type":   {"application/octet-stream"},
		"Server":         {"sv1"},
	}, (map[string][]string)(resp.Header)))
}

func TestProxy_RecvHeaderError(t *testing.T) {
	api, k8sClient, client, req, requestCount := setupProxy(t)
	requestCount.EXPECT().Inc()
	mrClient := mock_kubernetes_api.NewMockKubernetesApi_MakeRequestClient(gomock.NewController(t))
	var reqCtx context.Context
	mrCall := k8sClient.EXPECT().
		MakeRequest(gomock.Any()).
		DoAndReturn(func(ctx context.Context, opts ...grpc.CallOption) (rpc.KubernetesApi_MakeRequestClient, error) {
			reqCtx = ctx
			return mrClient, nil
		})
	gomock.InOrder(
		mrClient.EXPECT().
			Send(gomock.Any()).
			DoAndReturn(func(*grpctool.HttpRequest) error {
				<-reqCtx.Done() // wait for the receiving side to return error
				return errors.New("expected error 1")
			}),
		api.EXPECT().
			HandleSendError(gomock.Any(), gomock.Any(), matcher.ErrorEq("expected error 1")),
	)
	gomock.InOrder(
		mrCall,
		mrClient.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(&grpctool.HttpResponse{
				Message: &grpctool.HttpResponse_Header_{
					Header: &grpctool.HttpResponse_Header{
						Response: &prototool.HttpResponse{
							StatusCode: http.StatusOK,
						},
					},
				},
			})),
		mrClient.EXPECT().
			RecvMsg(gomock.Any()).
			Return(errors.New("expected error 2")),
		api.EXPECT().
			HandleProcessingError(gomock.Any(), gomock.Any(), gomock.Any(), matcher.ErrorEq("expected error 2")),
	)
	_, err := client.Do(req)               // nolint: bodyclose
	assert.True(t, errors.Is(err, io.EOF)) // dropped connection is io.EOF
}

func TestProxy_ErrorAfterHeaderWritten(t *testing.T) {
	api, k8sClient, client, req, requestCount := setupProxy(t)
	requestCount.EXPECT().Inc()
	mrClient := mock_kubernetes_api.NewMockKubernetesApi_MakeRequestClient(gomock.NewController(t))
	var reqCtx context.Context
	mrCall := k8sClient.EXPECT().
		MakeRequest(gomock.Any()).
		DoAndReturn(func(ctx context.Context, opts ...grpc.CallOption) (rpc.KubernetesApi_MakeRequestClient, error) {
			reqCtx = ctx
			return mrClient, nil
		})
	gomock.InOrder(
		mrClient.EXPECT().
			Send(gomock.Any()).
			DoAndReturn(func(*grpctool.HttpRequest) error {
				<-reqCtx.Done() // wait for the receiving side to return error
				return errors.New("expected error 1")
			}),
		api.EXPECT().
			HandleSendError(gomock.Any(), gomock.Any(), matcher.ErrorEq("expected error 1")),
	)
	gomock.InOrder(
		mrCall,
		mrClient.EXPECT().
			RecvMsg(gomock.Any()).
			Return(errors.New("expected error 2")),
		api.EXPECT().
			HandleProcessingError(gomock.Any(), gomock.Any(), gomock.Any(), matcher.ErrorEq("expected error 2")),
	)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.EqualValues(t, http.StatusBadGateway, resp.StatusCode)
}

func requireCorrectOutgoingMeta(t *testing.T, ctx context.Context) {
	md, _ := metadata.FromOutgoingContext(ctx)
	vals := md.Get(modserver.RoutingAgentIdMetadataKey)
	require.Len(t, vals, 1)
	agentId, err := strconv.ParseInt(vals[0], 10, 64)
	require.NoError(t, err)
	require.Equal(t, testhelpers.AgentId, agentId)
}

func assertToken(t *testing.T, r *http.Request) bool {
	return assert.Equal(t, jobToken, r.Header.Get("Job-Token"))
}

func setupProxy(t *testing.T) (*mock_modserver.MockAPI, *mock_kubernetes_api.MockKubernetesApiClient, *http.Client, *http.Request, *mock_usage_metrics.MockCounter) {
	return setupProxyWithHandler(t, "", defaultGitLabHandler(t))
}

func defaultGitLabHandler(t *testing.T) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !assertToken(t, r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		testhelpers.RespondWithJSON(t, w, &jobInfo{
			AllowedAgents: []allowedAgent{
				{
					Id: testhelpers.AgentId,
				},
			},
			Job:      job{},
			Pipeline: pipeline{},
			Project:  project{},
			User:     user{},
		})
	}
}

func setupProxyWithHandler(t *testing.T, urlPathPrefix string, handler func(http.ResponseWriter, *http.Request)) (*mock_modserver.MockAPI, *mock_kubernetes_api.MockKubernetesApiClient, *http.Client, *http.Request, *mock_usage_metrics.MockCounter) {
	sv, err := grpctool.NewStreamVisitor(&grpctool.HttpResponse{})
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	mockApi := mock_modserver.NewMockAPI(ctrl)
	k8sClient := mock_kubernetes_api.NewMockKubernetesApiClient(ctrl)
	requestCount := mock_usage_metrics.NewMockCounter(ctrl)

	p := &kubernetesApiProxy{
		log:                 zaptest.NewLogger(t),
		api:                 mockApi,
		kubernetesApiClient: k8sClient,
		gitLabClient:        mock_gitlab.SetupClient(t, jobInfoApiPath, handler),
		streamVisitor:       sv,
		requestCount:        requestCount,
		serverName:          "sv1",
		urlPathPrefix:       urlPathPrefix,
	}
	listener := grpctool.NewDialListener()
	var wg wait.Group
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		wg.Wait()
		listener.Close()
	})
	wg.Start(func() {
		assert.NoError(t, p.Run(ctx, listener))
	})
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return listener.DialContext(ctx, addr)
			},
		},
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"http://any_host_will_do.local"+urlPathPrefix+requestPath+"?"+url.QueryEscape(queryParamName)+"="+url.QueryEscape(queryParamValue),
		strings.NewReader(requestPayload),
	)
	require.NoError(t, err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s:%d:%s", tokenTypeCi, testhelpers.AgentId, jobToken))
	return mockApi, k8sClient, client, req, requestCount
}

func mockRecvStream(server *mock_kubernetes_api.MockKubernetesApi_MakeRequestClient, msgs ...proto.Message) []*gomock.Call {
	var res []*gomock.Call
	for _, msg := range msgs {
		call := server.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(msg))
		res = append(res, call)
	}
	call := server.EXPECT().
		RecvMsg(gomock.Any()).
		Return(io.EOF)
	res = append(res, call)
	return res
}

func mockSendStream(t *testing.T, client *mock_kubernetes_api.MockKubernetesApi_MakeRequestClient, msgs ...*grpctool.HttpRequest) []*gomock.Call {
	var res []*gomock.Call
	for _, msg := range msgs {
		call := client.EXPECT().
			Send(matcher.ProtoEq(t, msg))
		res = append(res, call)
	}
	res = append(res, client.EXPECT().CloseSend())
	return res
}
