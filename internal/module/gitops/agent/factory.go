package agent

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/retry"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/provider"
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
	factory := util.NewFactory(config.K8sClientGetter)
	return &module{
		log: config.Log,
		workerFactory: &defaultGitopsWorkerFactory{
			log: config.Log,
			applierFactory: &defaultApplierFactory{
				provider: provider.NewProvider(factory),
			},
			k8sUtilFactory: factory,
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
