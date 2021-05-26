package server

import (
	"strings"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/prototool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
)

const (
	defaultKubernetesApiListenAddress = "0.0.0.0:8154"
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
}
