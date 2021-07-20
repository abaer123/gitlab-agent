package server

import (
	"context"
	"io"
	"net/http"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
	gapi "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitlab_access/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
)

type server struct {
	rpc.UnimplementedGitlabAccessServer
	pipe *grpctool.InboundGrpcToOutboundHttp
}

func newServer(serverApi modserver.API, gitLabClient gitlab.ClientInterface) *server {
	return &server{
		pipe: grpctool.NewInboundGrpcToOutboundHttp(
			serverApi,
			func(ctx context.Context, header *grpctool.HttpRequest_Header, body io.Reader) (*http.Response, error) {
				var extra rpc.HeaderExtra
				err := header.Extra.UnmarshalTo(&extra)
				if err != nil {
					return nil, err
				}
				return gapi.MakeModuleRequest(
					ctx,
					gitLabClient,
					extra.ModuleName,
					header.Request.Method,
					header.Request.UrlPath,
					header.Request.UrlQuery(),
					header.Request.HttpHeader(),
					body,
				)
			},
		),
	}
}

func (s *server) MakeRequest(server rpc.GitlabAccess_MakeRequestServer) error {
	return s.pipe.Pipe(server)
}
