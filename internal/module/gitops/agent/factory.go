package agent

import (
	"fmt"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/retry"
)

const (
	getObjectsToSynchronizeInitBackoff   = 10 * time.Second
	getObjectsToSynchronizeMaxBackoff    = 5 * time.Minute
	getObjectsToSynchronizeResetDuration = 10 * time.Minute
	getObjectsToSynchronizeBackoffFactor = 2.0
	getObjectsToSynchronizeJitter        = 1.0

	engineInitBackoff   = 10 * time.Second
	engineMaxBackoff    = time.Minute
	engineResetDuration = time.Minute
	engineBackoffFactor = 2.0
	engineJitter        = 1.0
)

type Factory struct {
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
			k8sClientGetter: config.K8sClientGetter,
			watchBackoffFactory: retry.NewExponentialBackoffFactory(
				getObjectsToSynchronizeInitBackoff,
				getObjectsToSynchronizeMaxBackoff,
				getObjectsToSynchronizeResetDuration,
				getObjectsToSynchronizeBackoffFactor,
				getObjectsToSynchronizeJitter,
			),
			engineBackoffFactory: retry.NewExponentialBackoffFactory(
				engineInitBackoff,
				engineMaxBackoff,
				engineResetDuration,
				engineBackoffFactor,
				engineJitter,
			),
			gitopsClient: rpc.NewGitopsClient(config.KasConn),
		},
	}, nil
}

func (f *Factory) Name() string {
	return gitops.ModuleName
}
