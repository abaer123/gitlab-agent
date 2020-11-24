package gitops_agent

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/modclient"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/argoproj/gitops-engine/pkg/engine"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest"
)

const (
	ModuleName = "gitops"
)

type GitOpsEngineFactory interface {
	New(engineOpts []engine.Option, cacheOpts []cache.UpdateSettingsFunc) engine.GitOpsEngine
}

type Module struct {
	log                                *zap.Logger
	engineFactory                      GitOpsEngineFactory
	k8sClientGetter                    resource.RESTClientGetter
	getObjectsToSynchronizeRetryPeriod time.Duration
	api                                modclient.AgentAPI
	workers                            map[string]*gitopsWorkerHolder // project id -> worker holder instance
}

func (m *Module) Run(ctx context.Context) error {
	defer m.stopAllWorkers()
	<-ctx.Done()
	return nil
}

func (m *Module) DefaultAndValidateConfiguration(config *agentcfg.AgentConfiguration) error {
	protodefault.NotNil(&config.Gitops)
	return nil
}

func (m *Module) SetConfiguration(config *agentcfg.AgentConfiguration) error {
	err := m.configureWorkers(config.Gitops.ManifestProjects)
	if err != nil {
		return fmt.Errorf("manifest projects: %v", err)
	}
	return nil
}

func (m *Module) Name() string {
	return ModuleName
}

func (m *Module) stopAllWorkers() {
	// Tell all workers to stop
	for _, workerHolder := range m.workers {
		workerHolder.stop()
	}
	// Wait for all workers to stop
	for _, workerHolder := range m.workers {
		workerHolder.wg.Wait()
	}
}

func (m *Module) startNewWorker(project *agentcfg.ManifestProjectCF) {
	l := m.log.With(logz.ProjectId(project.Id))
	l.Info("Starting synchronization worker")
	worker := &gitopsWorker{
		api:                                m.api,
		engineFactory:                      m.engineFactory,
		getObjectsToSynchronizeRetryPeriod: m.getObjectsToSynchronizeRetryPeriod,
		synchronizerConfig: synchronizerConfig{
			log:                  l,
			projectConfiguration: project,
			k8sClientGetter:      m.k8sClientGetter,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	workerHolder := &gitopsWorkerHolder{
		worker: worker,
		stop:   cancel,
	}
	workerHolder.wg.StartWithContext(ctx, worker.Run)
	m.workers[project.Id] = workerHolder
}

func (m *Module) configureWorkers(projects []*agentcfg.ManifestProjectCF) error {
	newSetOfProjects := sets.NewString()
	var (
		projectsToStartWorkersFor []*agentcfg.ManifestProjectCF
		workersToStop             []*gitopsWorkerHolder
	)

	// Collect projects without workers or with updated configuration.
	for _, project := range projects {
		if newSetOfProjects.Has(project.Id) {
			return fmt.Errorf("duplicate project id: %s", project.Id)
		}
		newSetOfProjects.Insert(project.Id)
		workerHolder := m.workers[project.Id]
		if workerHolder == nil { // New project added
			projectsToStartWorkersFor = append(projectsToStartWorkersFor, project)
		} else { // We have a worker for this project already
			if proto.Equal(project, workerHolder.worker.projectConfiguration) {
				// Worker's configuration hasn't changed, nothing to do here
				continue
			}
			m.log.Info("Configuration has been updated, restarting synchronization worker", logz.ProjectId(project.Id))
			workersToStop = append(workersToStop, workerHolder)
			projectsToStartWorkersFor = append(projectsToStartWorkersFor, project)
		}
	}

	// Stop workers for projects which have been removed from the list.
	for projectId, workerHolder := range m.workers {
		if newSetOfProjects.Has(projectId) {
			continue
		}
		workersToStop = append(workersToStop, workerHolder)
	}

	// Tell workers that should be stopped to stop.
	for _, workerHolder := range workersToStop {
		projectId := workerHolder.worker.projectConfiguration.Id
		m.log.Info("Stopping synchronization worker", logz.ProjectId(projectId))
		workerHolder.stop()
		delete(m.workers, projectId)
	}

	// Wait for stopped workers to finish.
	for _, workerHolder := range workersToStop {
		projectId := workerHolder.worker.projectConfiguration.Id
		m.log.Info("Waiting for synchronization worker to stop", logz.ProjectId(projectId))
		workerHolder.wg.Wait()
	}

	// Start new workers for new projects or because of updated configuration.
	for _, project := range projectsToStartWorkersFor {
		m.startNewWorker(project)
	}
	return nil
}

type gitopsWorkerHolder struct {
	worker *gitopsWorker
	wg     wait.Group
	stop   context.CancelFunc
}

type DefaultGitOpsEngineFactory struct {
	KubeClientConfig *rest.Config
}

func (f *DefaultGitOpsEngineFactory) New(engineOpts []engine.Option, cacheOpts []cache.UpdateSettingsFunc) engine.GitOpsEngine {
	return engine.NewEngine(
		f.KubeClientConfig,
		cache.NewClusterCache(f.KubeClientConfig, cacheOpts...),
		engineOpts...,
	)
}
