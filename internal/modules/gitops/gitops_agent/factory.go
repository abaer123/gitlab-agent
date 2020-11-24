package gitops_agent

import (
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/modclient"
	"go.uber.org/zap"
	"k8s.io/cli-runtime/pkg/resource"
)

type Factory struct {
	Log                                *zap.Logger
	EngineFactory                      GitOpsEngineFactory
	K8sClientGetter                    resource.RESTClientGetter
	GetObjectsToSynchronizeRetryPeriod time.Duration
}

func (f *Factory) New(api modclient.AgentAPI) modclient.Module {
	return &Module{
		log:                                f.Log,
		engineFactory:                      f.EngineFactory,
		k8sClientGetter:                    f.K8sClientGetter,
		getObjectsToSynchronizeRetryPeriod: f.GetObjectsToSynchronizeRetryPeriod,
		api:                                api,
		workers:                            make(map[string]*gitopsWorkerHolder),
	}
}
