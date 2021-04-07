package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/kubernetes_api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/kubernetes_api/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	pipe *grpctool.InboundGrpcToOutboundHttp
}

func newModule(userAgent string, client httpClient, baseUrl *url.URL) *module {
	return &module{
		pipe: grpctool.NewInboundGrpcToOutboundHttp(
			handleProcessingError,
			handleSendError,
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

				resp, err := client.Do(req) // nolint: bodyclose
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

func handleSendError(log *zap.Logger, msg string, err error) error {
	// The problem is almost certainly with the client's connection.
	// Still log it on Debug.
	if !grpctool.RequestCanceled(err) {
		log.Debug(msg, zap.Error(err))
	}
	return status.Error(codes.Unavailable, "gRPC send failed")
}

func handleProcessingError(ctx context.Context, log *zap.Logger, msg string, err error) {
	if grpctool.RequestCanceled(err) {
		// An error caused by context signalling done
		return
	}
	var ue *errz.UserError
	isUserError := errors.As(err, &ue)
	if isUserError {
		// TODO Don't log it, send it somewhere the user can see it https://gitlab.com/gitlab-org/gitlab/-/issues/277323
		// Log at Info for now.
		log.Info(msg, zap.Error(err))
	} else {
		log.Error(msg, zap.Error(err))
	}
}
