package agent

import (
	"fmt"

	"github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/cilium_alert"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
)

type Factory struct {
}

func (f *Factory) New(cfg *modagent.Config) (modagent.Module, error) {
	restConfig, err := cfg.K8sClientGetter.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	ciliumClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("client set for cilium v2: %v", err)
	}
	return &module{
		log:          cfg.Log,
		api:          cfg.Api,
		ciliumClient: ciliumClient,
	}, nil
}

func (f *Factory) Name() string {
	return cilium_alert.ModuleName
}
