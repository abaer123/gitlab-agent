package server

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitlab_access/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_rpc"
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
	gitLabClient := mock_gitlab.NewMockClientInterface(ctrl)
	server := mock_rpc.NewMockGitlabAccess_MakeRequestServer(ctrl)
	incomingCtx := mock_modserver.IncomingCtx(ctx, t, token)
	agentMD, err := grpctool.AgentMDFromRawContext(incomingCtx)
	require.NoError(t, err)
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
			Message: &rpc.Request_Headers_{
				Headers: &rpc.Request_Headers{
					ModuleName: moduleName,
					Method:     httpMethod,
					Headers:    rpc.HeadersFromHttpHeaders(header),
					UrlPath:    urlPath,
					Query:      rpc.QueryFromUrlValues(query),
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
			Message: &rpc.Request_Trailers_{
				Trailers: &rpc.Request_Trailers{},
			},
		},
	)...)
	respHeader := http.Header{
		"resp": []string{"r1", "r2"},
	}
	respBody := []byte("some response")
	var wg sync.WaitGroup
	defer wg.Wait()
	pr, pw := io.Pipe()
	doStream := gitLabClient.EXPECT().
		DoStream(gomock.Any(), httpMethod, "/api/v4/internal/kubernetes/modules/"+moduleName+urlPath, header, query, agentMD.Token, gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, headers http.Header, query url.Values, agentToken api.AgentToken, body io.Reader) (*http.Response, error) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer pw.Close() // close the write side of the pipe
				all, errIO := ioutil.ReadAll(body)
				if !assert.NoError(t, errIO) {
					return
				}
				assert.Equal(t, []byte{1, 2, 3, 4, 5, 6}, all)
				_, errIO = pw.Write(respBody) // only respond once the request is consumed
				assert.NoError(t, errIO)
			}()
			return &http.Response{
				Status:     "status1",
				StatusCode: 200,
				Header:     respHeader,
				Body:       pr,
			}, nil
		})
	responses := mockSendStream(t, server,
		&rpc.Response{
			Message: &rpc.Response_Headers_{
				Headers: &rpc.Response_Headers{
					StatusCode: 200,
					Status:     "status1",
					Headers:    rpc.HeadersFromHttpHeaders(respHeader),
				}},
		},
		&rpc.Response{
			Message: &rpc.Response_Data_{
				Data: &rpc.Response_Data{
					Data: respBody,
				}},
		},
		&rpc.Response{
			Message: &rpc.Response_Trailers_{
				Trailers: &rpc.Response_Trailers{},
			},
		},
	)
	gomock.InOrder(append([]*gomock.Call{doStream}, responses...)...)
	f := Factory{}
	m, err := f.New(&modserver.Config{
		Log:          zaptest.NewLogger(t),
		Api:          mockApi,
		GitLabClient: gitLabClient,
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
			Do(mock_rpc.RetMsg(msg))
		res = append(res, call)
	}
	if eof {
		call := server.EXPECT().
			RecvMsg(gomock.Any()).
			DoAndReturn(func(msg interface{}) error {
				return io.EOF
			})
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
