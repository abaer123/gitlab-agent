package agent

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/util/wait"
)

type workerManager struct {
	log           *zap.Logger
	workerFactory GitopsWorkerFactory
	workers       map[string]*gitopsWorkerHolder // project id -> worker holder instance
}

func newWorkerManager(log *zap.Logger, workerFactory GitopsWorkerFactory) *workerManager {
	return &workerManager{
		log:           log,
		workerFactory: workerFactory,
		workers:       map[string]*gitopsWorkerHolder{},
	}
}

func (m *workerManager) startNewWorker(agentId int64, project *agentcfg.ManifestProjectCF) {
	l := m.log.With(logz.ProjectId(project.Id))
	l.Info("Starting synchronization worker")
	worker := m.workerFactory.New(agentId, project)
	ctx, cancel := context.WithCancel(context.Background())
	workerHolder := &gitopsWorkerHolder{
		worker:  worker,
		project: project,
		stop:    cancel,
	}
	workerHolder.wg.StartWithContext(ctx, worker.Run)
	m.workers[project.Id] = workerHolder
}

func (m *workerManager) ApplyConfiguration(agentId int64, gitops *agentcfg.GitopsCF) error {
	projects := gitops.ManifestProjects
	newSetOfProjects := make(map[string]struct{}, len(projects))
	var projectsToStartWorkersFor []*agentcfg.ManifestProjectCF
	var workersToStop []*gitopsWorkerHolder

	// Collect projects without workers or with updated configuration.
	for _, project := range projects {
		if _, ok := newSetOfProjects[project.Id]; ok {
			return fmt.Errorf("duplicate project id: %s", project.Id)
		}
		newSetOfProjects[project.Id] = struct{}{}
		workerHolder := m.workers[project.Id]
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
	for projectId, workerHolder := range m.workers {
		if _, ok := newSetOfProjects[projectId]; ok {
			continue
		}
		workersToStop = append(workersToStop, workerHolder)
	}

	// Tell workers that should be stopped to stop.
	for _, workerHolder := range workersToStop {
		m.log.Info("Stopping synchronization worker", logz.ProjectId(workerHolder.project.Id))
		workerHolder.stop()
		delete(m.workers, workerHolder.project.Id)
	}

	// Wait for stopped workers to finish.
	for _, workerHolder := range workersToStop {
		m.log.Info("Waiting for synchronization worker to stop", logz.ProjectId(workerHolder.project.Id))
		workerHolder.wg.Wait()
	}

	// Start new workers for new projects or because of updated configuration.
	for _, project := range projectsToStartWorkersFor {
		m.startNewWorker(agentId, project)
	}
	return nil
}

func (m *workerManager) stopAllWorkers() {
	// Tell all workers to stop
	for _, workerHolder := range m.workers {
		workerHolder.stop()
	}
	// Wait for all workers to stop
	for _, workerHolder := range m.workers {
		workerHolder.wg.Wait()
	}
}

type gitopsWorkerHolder struct {
	worker  GitopsWorker
	project *agentcfg.ManifestProjectCF
	wg      wait.Group
	stop    context.CancelFunc
}
