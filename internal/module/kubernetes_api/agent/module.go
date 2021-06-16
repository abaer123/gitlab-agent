package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/kubernetes_api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/kubernetes_api/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
)

const (
	userAgentHeaderName = "User-Agent"
)

var (
	_ modagent.Module  = &module{}
	_ modagent.Factory = &Factory{}
)

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type module struct {
	rpc.UnimplementedKubernetesApiServer
	api  modagent.API
	pipe *grpctool.InboundGrpcToOutboundHttp
}

func newModule(api modagent.API, userAgent string, client httpClient, baseUrl *url.URL) *module {
	return &module{
		api: api,
		pipe: grpctool.NewInboundGrpcToOutboundHttp(
			api,
			func(ctx context.Context, h *grpctool.HttpRequest_Header, body io.Reader) (*http.Response, error) {
				u := *baseUrl
				u.Path = h.Request.UrlPath
				u.RawQuery = h.Request.UrlQuery().Encode()

				req, err := http.NewRequestWithContext(ctx, h.Request.Method, u.String(), body)
				if err != nil {
					return nil, err
				}
				req.Header = h.Request.HttpHeader()
				ua := req.Header.Get(userAgentHeaderName)
				if ua == "" {
					ua = userAgent
				} else {
					ua = fmt.Sprintf("%s via %s", ua, userAgent)
				}
				req.Header.Set(userAgentHeaderName, ua)

				resp, err := client.Do(req)
				if err != nil {
					select {
					case <-ctx.Done(): // assume request errored out because of context
						return nil, ctx.Err()
					default:
						return nil, err
					}
				}
				return resp, nil
			},
		),
	}
}

func (m *module) Run(ctx context.Context, cfg <-chan *agentcfg.AgentConfiguration) error {
	// The tunnel feature is always required because CI for the agent's configuration project
	// can always access the agent.
	m.api.ToggleFeature(modagent.Tunnel, true)
	return nil
}

func (m *module) DefaultAndValidateConfiguration(config *agentcfg.AgentConfiguration) error {
	return nil
}

func (m *module) Name() string {
	return kubernetes_api.ModuleName
}

func (m *module) MakeRequest(server rpc.KubernetesApi_MakeRequestServer) error {
	return m.pipe.Pipe(server)
}
