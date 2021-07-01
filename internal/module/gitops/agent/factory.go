package agent

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"sigs.k8s.io/cli-utils/pkg/util/factory"
)

const (
	getObjectsToSynchronizeInitBackoff   = 10 * time.Second
	getObjectsToSynchronizeMaxBackoff    = 5 * time.Minute
	getObjectsToSynchronizeResetDuration = 10 * time.Minute
	getObjectsToSynchronizeBackoffFactor = 2.0
	getObjectsToSynchronizeJitter        = 1.0

	applierInitBackoff   = 10 * time.Second
	applierMaxBackoff    = time.Minute
	applierResetDuration = time.Minute
	applierBackoffFactor = 2.0
	applierJitter        = 1.0
)

type Factory struct {
}

func (f *Factory) New(config *modagent.Config) (modagent.Module, error) {
	statusPoller, err := factory.NewStatusPoller(config.K8sUtilFactory)
	if err != nil {
		return nil, err
	}
	return &module{
		log: config.Log,
		workerFactory: &defaultGitopsWorkerFactory{
			log: config.Log,
			applierFactory: &defaultApplierFactory{
				factory:      config.K8sUtilFactory,
				statusPoller: statusPoller,
			},
			k8sUtilFactory: config.K8sUtilFactory,
			gitopsClient:   rpc.NewGitopsClient(config.KasConn),
			watchBackoffFactory: retry.NewExponentialBackoffFactory(
				getObjectsToSynchronizeInitBackoff,
				getObjectsToSynchronizeMaxBackoff,
				getObjectsToSynchronizeResetDuration,
				getObjectsToSynchronizeBackoffFactor,
				getObjectsToSynchronizeJitter,
			),
			applierBackoffFactory: retry.NewExponentialBackoffFactory(
				applierInitBackoff,
				applierMaxBackoff,
				applierResetDuration,
				applierBackoffFactor,
				applierJitter,
			),
		},
	}, nil
}

func (f *Factory) Name() string {
	return gitops.ModuleName
}
