package server

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitlab_access/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/prototool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/testhelpers"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	_ rpc.GitlabAccessServer = &server{}
)

const (
	httpMethod = http.MethodPost
	urlPath    = "/bla"
	moduleName = "mod1"
)

func TestMakeRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockApi := mock_modserver.NewMockAPI(ctrl)
	server := mock_rpc.NewMockGitlabAccess_MakeRequestServer(ctrl)
	incomingCtx := mock_modserver.IncomingCtx(context.Background(), t, testhelpers.AgentkToken)
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
	extra, err := anypb.New(&rpc.HeaderExtra{
		ModuleName: moduleName,
	})
	require.NoError(t, err)
	gomock.InOrder(mockRecvStream(server, true,
		&grpctool.HttpRequest{
			Message: &grpctool.HttpRequest_Header_{
				Header: &grpctool.HttpRequest_Header{
					Request: &prototool.HttpRequest{
						Method:  httpMethod,
						Header:  prototool.HttpHeaderToValuesMap(header),
						UrlPath: urlPath,
						Query:   prototool.UrlValuesToValuesMap(query),
					},
					Extra: extra,
				},
			},
		},
		&grpctool.HttpRequest{
			Message: &grpctool.HttpRequest_Data_{
				Data: &grpctool.HttpRequest_Data{
					Data: []byte{1, 2, 3},
				},
			},
		},
		&grpctool.HttpRequest{
			Message: &grpctool.HttpRequest_Data_{
				Data: &grpctool.HttpRequest_Data{
					Data: []byte{4, 5, 6},
				},
			},
		},
		&grpctool.HttpRequest{
			Message: &grpctool.HttpRequest_Trailer_{
				Trailer: &grpctool.HttpRequest_Trailer{},
			},
		},
	)...)
	respBody := []byte("some response")
	gomock.InOrder(mockSendStream(t, server,
		&grpctool.HttpResponse{
			Message: &grpctool.HttpResponse_Header_{
				Header: &grpctool.HttpResponse_Header{
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
		&grpctool.HttpResponse{
			Message: &grpctool.HttpResponse_Data_{
				Data: &grpctool.HttpResponse_Data{
					Data: respBody,
				},
			},
		},
		&grpctool.HttpResponse{
			Message: &grpctool.HttpResponse_Trailer_{
				Trailer: &grpctool.HttpResponse_Trailer{},
			},
		},
	)...)
	s := newServer(mockApi, mock_gitlab.SetupClient(t, "/api/v4/internal/kubernetes/modules/"+moduleName+urlPath, func(w http.ResponseWriter, r *http.Request) {
		all, errIO := io.ReadAll(r.Body)
		if !assert.NoError(t, errIO) {
			return
		}
		assert.Equal(t, []byte{1, 2, 3, 4, 5, 6}, all)
		w.Header().Set("resp", "r1")
		w.Header().Add("resp", "r2")
		w.Header().Set("Date", "no date") // override
		_, errIO = w.Write(respBody)      // only respond once the request is consumed
		assert.NoError(t, errIO)
	}))
	require.NoError(t, s.MakeRequest(server))
}

func mockRecvStream(server *mock_rpc.MockGitlabAccess_MakeRequestServer, eof bool, msgs ...proto.Message) []*gomock.Call {
	res := make([]*gomock.Call, 0, len(msgs)+1)
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

func mockSendStream(t *testing.T, server *mock_rpc.MockGitlabAccess_MakeRequestServer, msgs ...*grpctool.HttpResponse) []*gomock.Call {
	res := make([]*gomock.Call, 0, len(msgs))
	for _, msg := range msgs {
		call := server.EXPECT().
			SendMsg(matcher.ProtoEq(t, msg))
		res = append(res, call)
	}
	return res
}
