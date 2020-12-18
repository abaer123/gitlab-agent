package agent

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
)

type Factory struct {
	EngineFactory                      GitOpsEngineFactory
	GetObjectsToSynchronizeRetryPeriod time.Duration
}

func (f *Factory) New(config *modagent.Config) (modagent.Module, error) {
	return &module{
		log:                                config.Log,
		engineFactory:                      f.EngineFactory,
		k8sClientGetter:                    config.K8sClientGetter,
		getObjectsToSynchronizeRetryPeriod: f.GetObjectsToSynchronizeRetryPeriod,
		gitopsClient:                       rpc.NewGitopsClient(config.KasConn),
		workers:                            make(map[string]*gitopsWorkerHolder),
	}, nil
}

func (f *Factory) Name() string {
	return gitops.ModuleName
}
