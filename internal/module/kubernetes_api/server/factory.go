package server

import (
	"crypto/tls"
	"fmt"
	"net"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/kubernetes_api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/kubernetes_api/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/cache"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/tlstool"
)

const (
	k8sApiRequestCountKnownMetric = "k8s_api_proxy_request_count"
)

type Factory struct {
}

func (f *Factory) New(config *modserver.Config) (modserver.Module, error) {
	k8sApi := config.Config.Agent.KubernetesApi
	if k8sApi == nil {
		return nopModule{}, nil
	}
	sv, err := grpctool.NewStreamVisitor(&grpctool.HttpResponse{})
	if err != nil {
		return nil, err
	}
	listenCfg := k8sApi.Listen
	certFile := listenCfg.CertificateFile
	keyFile := listenCfg.KeyFile
	var listener func() (net.Listener, error)

	switch {
	case certFile != "" && keyFile != "":
		tlsConfig, err := tlstool.DefaultServerTLSConfig(certFile, keyFile)
		if err != nil {
			return nil, err
		}
		listener = func() (net.Listener, error) {
			return tls.Listen(listenCfg.Network.String(), listenCfg.Address, tlsConfig)
		}
	case certFile == "" && keyFile == "":
		listener = func() (net.Listener, error) {
			return net.Listen(listenCfg.Network.String(), listenCfg.Address)
		}
	default:
		return nil, fmt.Errorf("both certificate_file (%s) and key_file (%s) must be either set or not set", certFile, keyFile)
	}
	m := &module{
		log: config.Log,
		proxy: kubernetesApiProxy{
			log:                 config.Log,
			api:                 config.Api,
			kubernetesApiClient: rpc.NewKubernetesApiClient(config.AgentConn),
			gitLabClient:        config.GitLabClient,
			streamVisitor:       sv,
			cache:               cache.NewWithError(k8sApi.AllowedAgentCacheTtl.AsDuration(), k8sApi.AllowedAgentCacheErrorTtl.AsDuration()),
			requestCount:        config.UsageTracker.RegisterCounter(k8sApiRequestCountKnownMetric),
			serverName:          fmt.Sprintf("%s/%s/%s", config.KasName, config.Version, config.CommitId),
			urlPathPrefix:       k8sApi.UrlPathPrefix,
		},
		listener: listener,
	}
	config.RegisterAgentApi(&rpc.KubernetesApi_ServiceDesc)
	return m, nil
}

func (f *Factory) Name() string {
	return kubernetes_api.ModuleName
}
