package server

import (
	"strings"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/prototool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/kascfg"
)

const (
	defaultKubernetesApiListenAddress    = "0.0.0.0:8154"
	defaultAllowedAgentInfoCacheTTL      = 1 * time.Minute
	defaultAllowedAgentInfoCacheErrorTTL = 10 * time.Second
)

func ApplyDefaults(config *kascfg.ConfigurationFile) {
	prototool.NotNil(&config.Agent)
	o := config.Agent.KubernetesApi

	if o == nil {
		return
	}
	prototool.NotNil(&o.Listen)
	prototool.String(&o.Listen.Address, defaultKubernetesApiListenAddress)
	if !strings.HasSuffix(o.UrlPathPrefix, "/") {
		o.UrlPathPrefix = o.UrlPathPrefix + "/"
	}
	prototool.Duration(&o.AllowedAgentCacheTtl, defaultAllowedAgentInfoCacheTTL)
	prototool.Duration(&o.AllowedAgentCacheErrorTtl, defaultAllowedAgentInfoCacheErrorTTL)
}
