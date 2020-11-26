package gitops_agent

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/modagent"
)

type Factory struct {
	EngineFactory                      GitOpsEngineFactory
	GetObjectsToSynchronizeRetryPeriod time.Duration
}

func (f *Factory) New(config *modagent.Config) modagent.Module {
	return &module{
		log:                                config.Log,
		engineFactory:                      f.EngineFactory,
		k8sClientGetter:                    config.K8sClientGetter,
		getObjectsToSynchronizeRetryPeriod: f.GetObjectsToSynchronizeRetryPeriod,
		kasClient:                          agentrpc.NewKasClient(config.KasConn),
		workers:                            make(map[string]*gitopsWorkerHolder),
	}
}
