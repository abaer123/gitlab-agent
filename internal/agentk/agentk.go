package agentk

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/argoproj/gitops-engine/pkg/engine"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"gitlab.com/gitlab-org/labkit/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest"
)

const (
	defaultRefreshConfigurationRetryPeriod    = 10 * time.Second
	defaultGetObjectsToSynchronizeRetryPeriod = 10 * time.Second

	defaultNamespace = metav1.NamespaceDefault
)

type GitOpsEngineFactory interface {
	New(...cache.UpdateSettingsFunc) engine.GitOpsEngine
}

type Agent struct {
	kasClient                       agentrpc.KasClient
	engineFactory                   GitOpsEngineFactory
	k8sClientGetter                 resource.RESTClientGetter
	workers                         map[string]*gitopsWorkerHolder // project id -> worker holder instance
	refreshConfigurationRetryPeriod time.Duration
}

type gitopsWorkerHolder struct {
	worker *gitopsWorker
	wg     wait.Group
	stop   context.CancelFunc
}

func New(kasClient agentrpc.KasClient, engineFactory GitOpsEngineFactory, k8sClientGetter resource.RESTClientGetter) *Agent {
	return &Agent{
		kasClient:                       kasClient,
		engineFactory:                   engineFactory,
		k8sClientGetter:                 k8sClientGetter,
		workers:                         make(map[string]*gitopsWorkerHolder),
		refreshConfigurationRetryPeriod: defaultRefreshConfigurationRetryPeriod,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	defer a.stopAllWorkers()
	retry.JitterUntil(ctx, a.refreshConfigurationRetryPeriod, a.refreshConfiguration())
	return nil
}

func (a *Agent) stopAllWorkers() {
	// Tell all workers to stop
	for _, workerHolder := range a.workers {
		workerHolder.stop()
	}
	// Wait for all workers to stop
	for _, workerHolder := range a.workers {
		workerHolder.wg.Wait()
	}
}

func (a *Agent) refreshConfiguration() func(context.Context) {
	var lastProcessedCommitId string
	return func(ctx context.Context) {
		req := &agentrpc.ConfigurationRequest{
			CommitId: lastProcessedCommitId,
		}
		res, err := a.kasClient.GetConfiguration(ctx, req)
		if err != nil {
			log.WithError(err).Warn("GetConfiguration failed")
			return
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
				return
			}
			lastProcessedCommitId = config.CommitId
			err = a.applyConfiguration(config.Configuration)
			if err != nil {
				log.WithError(err).Error("Failed to apply configuration")
			}
		}
	}
}

func (a *Agent) applyConfiguration(config *agentcfg.AgentConfiguration) error {
	log.WithField("config", config).Debug("Applying configuration")
	err := a.applyGitOpsConfiguration(config.Gitops)
	if err != nil {
		return fmt.Errorf("gitops: %v", err)
	}
	return nil
}

func (a *Agent) applyGitOpsConfiguration(gitops *agentcfg.GitopsCF) error {
	err := a.configureWorkers(gitops.GetManifestProjects())
	if err != nil {
		return fmt.Errorf("manifest projects: %v", err)
	}
	return nil
}

func (a *Agent) configureWorkers(projects []*agentcfg.ManifestProjectCF) error {
	newSetOfProjects := sets.NewString()
	var (
		projectsToStartWorkersFor []*agentcfg.ManifestProjectCF
		workersToStop             []*gitopsWorkerHolder
	)

	// Collect projects without workers or with updated configuration.
	for _, project := range projects {
		if newSetOfProjects.Has(project.Id) {
			log.WithField(api.ProjectId, project.Id).Error()
			return fmt.Errorf("duplicate project id: %s", project.Id)
		}
		applyDefaultsToManifestProject(project)
		newSetOfProjects.Insert(project.Id)
		workerHolder := a.workers[project.Id]
		if workerHolder == nil { // New project added
			projectsToStartWorkersFor = append(projectsToStartWorkersFor, project)
		} else { // We have a worker for this project already
			if proto.Equal(project, workerHolder.worker.projectConfiguration) {
				// Worker's configuration hasn't changed, nothing to do here
				continue
			}
			log.WithField(api.ProjectId, project.Id).Info("Configuration has been updated, restarting synchronization worker")
			workersToStop = append(workersToStop, workerHolder)
			projectsToStartWorkersFor = append(projectsToStartWorkersFor, project)
		}
	}

	// Stop workers for projects which have been removed from the list.
	for projectId, workerHolder := range a.workers {
		if newSetOfProjects.Has(projectId) {
			continue
		}
		workersToStop = append(workersToStop, workerHolder)
	}

	// Tell workers that should be stopped to stop.
	for _, workerHolder := range workersToStop {
		projectId := workerHolder.worker.projectConfiguration.Id
		log.WithField(api.ProjectId, projectId).Info("Stopping synchronization worker")
		workerHolder.stop()
		delete(a.workers, projectId)
	}

	// Wait for stopped workers to finish.
	for _, workerHolder := range workersToStop {
		projectId := workerHolder.worker.projectConfiguration.Id
		log.WithField(api.ProjectId, projectId).Info("Waiting for synchronization worker to stop")
		workerHolder.wg.Wait()
	}

	// Start new workers for new projects or because of updated configuration.
	for _, project := range projectsToStartWorkersFor {
		a.startNewWorker(project)
	}
	return nil
}

func (a *Agent) startNewWorker(project *agentcfg.ManifestProjectCF) {
	logger := log.WithField(api.ProjectId, project.Id)
	logger.Info("Starting synchronization worker")
	worker := &gitopsWorker{
		kasClient:                          a.kasClient,
		engineFactory:                      a.engineFactory,
		getObjectsToSynchronizeRetryPeriod: defaultGetObjectsToSynchronizeRetryPeriod,
		synchronizerConfig: synchronizerConfig{
			log:                  logger,
			projectConfiguration: project,
			k8sClientGetter:      a.k8sClientGetter,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	workerHolder := &gitopsWorkerHolder{
		worker: worker,
		stop:   cancel,
	}
	workerHolder.wg.StartWithContext(ctx, worker.Run)
	a.workers[project.Id] = workerHolder
}

type DefaultGitOpsEngineFactory struct {
	KubeClientConfig *rest.Config
}

func (f *DefaultGitOpsEngineFactory) New(opts ...cache.UpdateSettingsFunc) engine.GitOpsEngine {
	return engine.NewEngine(f.KubeClientConfig, cache.NewClusterCache(f.KubeClientConfig, opts...))
}

func applyDefaultsToManifestProject(project *agentcfg.ManifestProjectCF) {
	if project.DefaultNamespace == "" {
		project.DefaultNamespace = defaultNamespace
	}
}
