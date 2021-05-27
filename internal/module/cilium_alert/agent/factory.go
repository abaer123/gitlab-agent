package agent

import (
	"fmt"
	"time"

	"github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/cilium_alert"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
)

const (
	getFlowsPollInterval = 10 * time.Second
	informerResyncPeriod = 30 * time.Minute

	pollingInitBackoff   = 10 * time.Second
	pollingMaxBackoff    = 5 * time.Minute
	pollingResetDuration = 10 * time.Minute
	pollingBackoffFactor = 2.0
	pollingJitter        = 1.0
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
		backoff: retry.NewExponentialBackoffFactory(
			pollingInitBackoff,
			pollingMaxBackoff,
			pollingResetDuration,
			pollingBackoffFactor,
			pollingJitter,
		),
		getFlowsPollInterval: getFlowsPollInterval,
		informerResyncPeriod: informerResyncPeriod,
	}, nil
}

func (f *Factory) Name() string {
	return cilium_alert.ModuleName
}
