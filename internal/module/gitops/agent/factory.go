package agent

import (
	"fmt"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
)

type Factory struct {
	GetObjectsToSynchronizeRetryPeriod time.Duration
}

func (f *Factory) New(config *modagent.Config) (modagent.Module, error) {
	restConfig, err := config.K8sClientGetter.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("ToRESTConfig: %v", err)
	}
	return &module{
		log: config.Log,
		workerFactory: &defaultGitopsWorkerFactory{
			log: config.Log,
			engineFactory: &defaultGitopsEngineFactory{
				kubeClientConfig: restConfig,
			},
			k8sClientGetter:                    config.K8sClientGetter,
			getObjectsToSynchronizeRetryPeriod: f.GetObjectsToSynchronizeRetryPeriod,
			gitopsClient:                       rpc.NewGitopsClient(config.KasConn),
		},
	}, nil
}

func (f *Factory) Name() string {
	return gitops.ModuleName
}
