package server

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitlab_access/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/prototool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

const (
	token api.AgentToken = "abfaasdfasdfasdf"
)

var (
	_ modserver.Module       = &module{}
	_ modserver.Factory      = &Factory{}
	_ rpc.GitlabAccessServer = &module{}
)

const (
	httpMethod = http.MethodPost
	urlPath    = "/bla"
	moduleName = "mod1"
)

func TestMakeRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctrl := gomock.NewController(t)
	mockApi := mock_modserver.NewMockAPI(ctrl)
	server := mock_rpc.NewMockGitlabAccess_MakeRequestServer(ctrl)
	incomingCtx := mock_modserver.IncomingCtx(ctx, t, token)
	server.EXPECT().
		Context().
		Return(incomingCtx).
		MinTimes(1)
	header := http.Header{
		"k": []string{"v1", "v2"},
	}
	query := url.Values{
		"k1": []string{"q1", "q2"},
	}
	gomock.InOrder(mockRecvStream(server, true,
		&rpc.Request{
			Message: &rpc.Request_Header_{
				Header: &rpc.Request_Header{
					ModuleName: moduleName,
					Request: &prototool.HttpRequest{
						Method:  httpMethod,
						Header:  prototool.HttpHeaderToValuesMap(header),
						UrlPath: urlPath,
						Query:   prototool.UrlValuesToValuesMap(query),
					},
				},
			},
		},
		&rpc.Request{
			Message: &rpc.Request_Data_{
				Data: &rpc.Request_Data{
					Data: []byte{1, 2, 3},
				},
			},
		},
		&rpc.Request{
			Message: &rpc.Request_Data_{
				Data: &rpc.Request_Data{
					Data: []byte{4, 5, 6},
				},
			},
		},
		&rpc.Request{
			Message: &rpc.Request_Trailer_{
				Trailer: &rpc.Request_Trailer{},
			},
		},
	)...)
	respBody := []byte("some response")
	r := http.NewServeMux()
	r.HandleFunc("/api/v4/internal/kubernetes/modules/"+moduleName+urlPath, func(w http.ResponseWriter, r *http.Request) {
		all, errIO := ioutil.ReadAll(r.Body)
		if !assert.NoError(t, errIO) {
			return
		}
		assert.Equal(t, []byte{1, 2, 3, 4, 5, 6}, all)
		w.Header().Set("resp", "r1")
		w.Header().Add("resp", "r2")
		w.Header().Set("Date", "no date") // override
		_, errIO = w.Write(respBody)      // only respond once the request is consumed
		assert.NoError(t, errIO)
	})
	s := httptest.NewServer(r)
	defer s.Close()

	u, err := url.Parse(s.URL)
	require.NoError(t, err)
	gomock.InOrder(mockSendStream(t, server,
		&rpc.Response{
			Message: &rpc.Response_Header_{
				Header: &rpc.Response_Header{
					Response: &prototool.HttpResponse{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Header: map[string]*prototool.Values{
							"Content-Length": {
								Value: []string{"13"},
							},
							"Content-Type": {
								Value: []string{"text/plain; charset=utf-8"},
							},
							"Date": {
								Value: []string{"no date"},
							},
							"Resp": {
								Value: []string{"r1", "r2"},
							},
						},
					},
				},
			},
		},
		&rpc.Response{
			Message: &rpc.Response_Data_{
				Data: &rpc.Response_Data{
					Data: respBody,
				},
			},
		},
		&rpc.Response{
			Message: &rpc.Response_Trailer_{
				Trailer: &rpc.Response_Trailer{},
			},
		},
	)...)
	f := Factory{}
	log := zaptest.NewLogger(t)
	m, err := f.New(&modserver.Config{
		Log:          log,
		Api:          mockApi,
		GitLabClient: gitlab.NewClient(u, []byte{1, 2, 3}, gitlab.WithLogger(log)),
		AgentServer:  grpc.NewServer(),
		ApiServer:    grpc.NewServer(),
	})
	require.NoError(t, err)
	require.NoError(t, m.(*module).MakeRequest(server))
}

func mockRecvStream(server *mock_rpc.MockGitlabAccess_MakeRequestServer, eof bool, msgs ...proto.Message) []*gomock.Call {
	var res []*gomock.Call
	for _, msg := range msgs {
		call := server.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(msg))
		res = append(res, call)
	}
	if eof {
		call := server.EXPECT().
			RecvMsg(gomock.Any()).
			Return(io.EOF)
		res = append(res, call)
	}
	return res
}

func mockSendStream(t *testing.T, server *mock_rpc.MockGitlabAccess_MakeRequestServer, msgs ...*rpc.Response) []*gomock.Call {
	var res []*gomock.Call
	for _, msg := range msgs {
		call := server.EXPECT().
			Send(matcher.ProtoEq(t, msg))
		res = append(res, call)
	}
	return res
}
