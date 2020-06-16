package agentk

import (
	"context"
	"io"
	"sync"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/api"
	"gitlab.com/gitlab-org/labkit/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
)

const (
	refreshConfigurationRetryPeriod = 10 * time.Second
)

type Agent struct {
	client           agentrpc.GitLabServiceClient
	kubeClientConfig *rest.Config

	workers     map[string]*deploymentWorkerHolder // project id -> worker holder instance
	workersLock sync.RWMutex
	workersWg   wait.Group
}

type deploymentWorkerHolder struct {
	worker *deploymentWorker
	stop   context.CancelFunc
}

func New(client agentrpc.GitLabServiceClient, kubeClientConfig *rest.Config) *Agent {
	return &Agent{
		client:           client,
		kubeClientConfig: kubeClientConfig,
		workers:          make(map[string]*deploymentWorkerHolder),
	}
}

func (a *Agent) Run(ctx context.Context) error {
	defer a.workersWg.Wait() // Wait for all workers to stop
	defer a.stopAllWorkers()
	err := wait.PollImmediateUntil(refreshConfigurationRetryPeriod, a.refreshConfiguration(ctx), ctx.Done())
	if err == wait.ErrWaitTimeout {
		return nil // all good, ctx is done
	}
	return err
}

func (a *Agent) stopAllWorkers() {
	for _, workerHolder := range a.workers {
		workerHolder.stop()
	}
}

func (a *Agent) refreshConfiguration(ctx context.Context) wait.ConditionFunc {
	return func() (bool /*done*/, error) {
		req := &agentrpc.ConfigurationRequest{}
		res, err := a.client.GetConfiguration(ctx, req)
		if err != nil {
			log.WithError(err).Warn("GetConfiguration failed")
			return false, nil // nil error to keep polling
		}
		for {
			config, err := res.Recv()
			if err != nil {
				switch {
				case err == io.EOF:
				case status.Code(err) == codes.DeadlineExceeded:
				case status.Code(err) == codes.Canceled:
				default:
					log.WithError(err).Warn("GetConfiguration.Recv failed")
				}
				return false, nil // nil error to keep polling
			}
			a.applyConfiguration(config.Configuration)
		}
	}
}

func (a *Agent) applyConfiguration(config *agentrpc.AgentConfiguration) {
	log.WithField("config", config).Debug("Applying configuration")
	a.applyDeploymentsConfiguration(config.Deployments)
}

func (a *Agent) applyDeploymentsConfiguration(deployments *agentcfg.DeploymentsCF) {
	var projects []*agentcfg.ManifestProjectCF
	if deployments != nil {
		projects = deployments.ManifestProjects
	}
	a.synchronizeWorkers(projects)
}

func (a *Agent) synchronizeWorkers(projects []*agentcfg.ManifestProjectCF) {
	a.workersLock.Lock()
	defer a.workersLock.Unlock()

	newSetOfProjects := sets.NewString()
	var projectsToAdd []*agentcfg.ManifestProjectCF

	// Collect projects without workers.
	for _, project := range projects {
		newSetOfProjects.Insert(project.Id)
		workerHolder := a.workers[project.Id]
		if workerHolder == nil {
			projectsToAdd = append(projectsToAdd, project)
		} else {
			// TODO update worker's configuration. Nothing currently, but e.g. credentials in the future
		}
	}

	// Stop workers for projects which have been removed from the list.
	for projectId, workerHolder := range a.workers {
		if !newSetOfProjects.Has(projectId) {
			log.WithField(api.ProjectId, projectId).Info("Stopping synchronization worker")
			workerHolder.stop()
			delete(a.workers, projectId)
		}
	}

	// Start workers for newly added projects.
	for _, project := range projectsToAdd {
		a.startNewWorkerLocked(project)
	}
}

func (a *Agent) startNewWorkerLocked(project *agentcfg.ManifestProjectCF) {
	logger := log.WithField(api.ProjectId, project.Id)
	logger.Info("Starting synchronization worker")
	worker := &deploymentWorker{
		deploymentWorkerConfig: deploymentWorkerConfig{
			log:       logger,
			projectId: project.Id,
			//namespace:
			client:           a.client,
			kubeClientConfig: a.kubeClientConfig,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.workersWg.StartWithContext(ctx, worker.Run)
	a.workers[project.Id] = &deploymentWorkerHolder{
		worker: worker,
		stop:   cancel,
	}
}
