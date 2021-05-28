package agent

import (
	"fmt"
	"net/http"
	"net/url"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/kubernetes_api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/kubernetes_api/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modagent"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type Factory struct {
}

func (f *Factory) New(config *modagent.Config) (modagent.Module, error) {
	restConfig, err := config.K8sUtilFactory.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	transport, err := rest.TransportFor(restConfig)
	if err != nil {
		return nil, err
	}
	baseUrl, _, err := defaultServerUrlFor(restConfig)
	if err != nil {
		return nil, err
	}
	userAgent := fmt.Sprintf("%s/%s/%s", config.AgentName, config.AgentMeta.Version, config.AgentMeta.CommitId)
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	m := newModule(config.Api, userAgent, client, baseUrl)
	rpc.RegisterKubernetesApiServer(config.Server, m)
	return m, nil
}

func (f *Factory) Name() string {
	return kubernetes_api.ModuleName
}

// This is a copy from k8s.io/client-go/rest/url_utils.go

// defaultServerUrlFor is shared between IsConfigTransportTLS and RESTClientFor. It
// requires Host and Version to be set prior to being called.
func defaultServerUrlFor(config *rest.Config) (*url.URL, string, error) {
	// TODO: move the default to secure when the apiserver supports TLS by default
	// config.Insecure is taken to mean "I want HTTPS but don't bother checking the certs against a CA."
	hasCA := len(config.CAFile) != 0 || len(config.CAData) != 0
	hasCert := len(config.CertFile) != 0 || len(config.CertData) != 0
	defaultTLS := hasCA || hasCert || config.Insecure
	host := config.Host
	if host == "" {
		host = "localhost"
	}

	if config.GroupVersion != nil {
		return rest.DefaultServerURL(host, config.APIPath, *config.GroupVersion, defaultTLS)
	}
	return rest.DefaultServerURL(host, config.APIPath, schema.GroupVersion{}, defaultTLS)
}
