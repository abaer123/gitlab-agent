package agent

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/prototool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	defaultGitOpsManifestNamespace = metav1.NamespaceDefault
	defaultGitOpsManifestPathGlob  = "**/*.{yaml,yml,json}"
)

type module struct {
	log           *zap.Logger
	workerFactory GitopsWorkerFactory
}

func (m *module) Run(ctx context.Context, cfg <-chan *agentcfg.AgentConfiguration) error {
	workers := make(map[string]*gitopsWorkerHolder) // project id -> worker holder instance
	defer stopAllWorkers(workers)
	for config := range cfg {
		err := m.configureWorkers(workers, config.Gitops.ManifestProjects)
		if err != nil {
			m.log.Error("Failed to apply manifest projects configuration", zap.Error(err))
			continue
		}
	}
	return nil
}

func (m *module) DefaultAndValidateConfiguration(config *agentcfg.AgentConfiguration) error {
	prototool.NotNil(&config.Gitops)
	for _, project := range config.Gitops.ManifestProjects {
		applyDefaultsToManifestProject(project)
	}
	return nil
}

func applyDefaultsToManifestProject(project *agentcfg.ManifestProjectCF) {
	prototool.String(&project.DefaultNamespace, defaultGitOpsManifestNamespace)
	if len(project.Paths) == 0 {
		project.Paths = []*agentcfg.PathCF{
			{
				Glob: defaultGitOpsManifestPathGlob,
			},
		}
	}
}

func (m *module) Name() string {
	return gitops.ModuleName
}

func (m *module) startNewWorker(workers map[string]*gitopsWorkerHolder, project *agentcfg.ManifestProjectCF) {
	l := m.log.With(logz.ProjectId(project.Id))
	l.Info("Starting synchronization worker")
	worker := m.workerFactory.New(project)
	ctx, cancel := context.WithCancel(context.Background())
	workerHolder := &gitopsWorkerHolder{
		worker:  worker,
		project: project,
		stop:    cancel,
	}
	workerHolder.wg.StartWithContext(ctx, worker.Run)
	workers[project.Id] = workerHolder
}

func (m *module) configureWorkers(workers map[string]*gitopsWorkerHolder, projects []*agentcfg.ManifestProjectCF) error {
	newSetOfProjects := make(map[string]struct{}, len(projects))
	var projectsToStartWorkersFor []*agentcfg.ManifestProjectCF
	var workersToStop []*gitopsWorkerHolder

	// Collect projects without workers or with updated configuration.
	for _, project := range projects {
		if _, ok := newSetOfProjects[project.Id]; ok {
			return fmt.Errorf("duplicate project id: %s", project.Id)
		}
		newSetOfProjects[project.Id] = struct{}{}
		workerHolder := workers[project.Id]
		if workerHolder == nil { // New project added
			projectsToStartWorkersFor = append(projectsToStartWorkersFor, project)
		} else { // We have a worker for this project already
			if proto.Equal(project, workerHolder.project) {
				// Worker's configuration hasn't changed, nothing to do here
				continue
			}
			m.log.Info("Configuration has been updated, restarting synchronization worker", logz.ProjectId(project.Id))
			workersToStop = append(workersToStop, workerHolder)
			projectsToStartWorkersFor = append(projectsToStartWorkersFor, project)
		}
	}

	// Stop workers for projects which have been removed from the list.
	for projectId, workerHolder := range workers {
		if _, ok := newSetOfProjects[projectId]; ok {
			continue
		}
		workersToStop = append(workersToStop, workerHolder)
	}

	// Tell workers that should be stopped to stop.
	for _, workerHolder := range workersToStop {
		m.log.Info("Stopping synchronization worker", logz.ProjectId(workerHolder.project.Id))
		workerHolder.stop()
		delete(workers, workerHolder.project.Id)
	}

	// Wait for stopped workers to finish.
	for _, workerHolder := range workersToStop {
		m.log.Info("Waiting for synchronization worker to stop", logz.ProjectId(workerHolder.project.Id))
		workerHolder.wg.Wait()
	}

	// Start new workers for new projects or because of updated configuration.
	for _, project := range projectsToStartWorkersFor {
		m.startNewWorker(workers, project)
	}
	return nil
}

func stopAllWorkers(workers map[string]*gitopsWorkerHolder) {
	// Tell all workers to stop
	for _, workerHolder := range workers {
		workerHolder.stop()
	}
	// Wait for all workers to stop
	for _, workerHolder := range workers {
		workerHolder.wg.Wait()
	}
}

type gitopsWorkerHolder struct {
	worker  GitopsWorker
	project *agentcfg.ManifestProjectCF
	wg      wait.Group
	stop    context.CancelFunc
}
