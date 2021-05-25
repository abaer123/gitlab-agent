package server

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitlab_access"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitlab_access/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
)

const (
	urlPathForModules = "/api/v4/internal/kubernetes/modules/"
)

type module struct {
	rpc.UnimplementedGitlabAccessServer
	pipe *grpctool.InboundGrpcToOutboundHttp
}

func newModule(serverApi modserver.API, gitLabClient gitlab.ClientInterface) *module {
	return &module{
		pipe: grpctool.NewInboundGrpcToOutboundHttp(
			serverApi,
			func(ctx context.Context, header *grpctool.HttpRequest_Header, body io.Reader) (*http.Response, error) {
				var extra rpc.HeaderExtra
				err := header.Extra.UnmarshalTo(&extra)
				if err != nil {
					return nil, err
				}
				var resp *http.Response
				err = gitLabClient.Do(ctx,
					gitlab.WithMethod(header.Request.Method),
					gitlab.WithPath(urlPathForModules+url.PathEscape(extra.ModuleName)+header.Request.UrlPath),
					gitlab.WithQuery(header.Request.UrlQuery()),
					gitlab.WithHeader(header.Request.HttpHeader()),
					gitlab.WithAgentToken(api.AgentTokenFromContext(ctx)),
					gitlab.WithJWT(true),
					gitlab.WithRequestBody(body, ""),
					gitlab.WithResponseHandler(gitlab.NakedResponseHandler(&resp)),
				)
				return resp, err
			},
		),
	}
}

func (m *module) MakeRequest(server rpc.GitlabAccess_MakeRequestServer) error {
	return m.pipe.Pipe(server)
}

func (m *module) Run(ctx context.Context) error {
	return nil
}

func (m *module) Name() string {
	return gitlab_access.ModuleName
}
