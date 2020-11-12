package agentk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/argoproj/gitops-engine/pkg/engine"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest"
)

type GitOpsEngineFactory interface {
	New(engineOpts []engine.Option, cacheOpts []cache.UpdateSettingsFunc) engine.GitOpsEngine
}

type Config struct {
	Log                                *zap.Logger
	LogLevel                           zap.AtomicLevel
	KasClient                          agentrpc.KasClient
	EngineFactory                      GitOpsEngineFactory
	K8sClientGetter                    resource.RESTClientGetter
	RefreshConfigurationRetryPeriod    time.Duration
	GetObjectsToSynchronizeRetryPeriod time.Duration
}

type Agent struct {
	log                                *zap.Logger
	logLevel                           zap.AtomicLevel
	kasClient                          agentrpc.KasClient
	engineFactory                      GitOpsEngineFactory
	k8sClientGetter                    resource.RESTClientGetter
	refreshConfigurationRetryPeriod    time.Duration
	getObjectsToSynchronizeRetryPeriod time.Duration
	workers                            map[string]*gitopsWorkerHolder // project id -> worker holder instance
}

type gitopsWorkerHolder struct {
	worker *gitopsWorker
	wg     wait.Group
	stop   context.CancelFunc
}

func New(config Config) *Agent {
	return &Agent{
		log:                                config.Log,
		logLevel:                           config.LogLevel,
		kasClient:                          config.KasClient,
		engineFactory:                      config.EngineFactory,
		k8sClientGetter:                    config.K8sClientGetter,
		refreshConfigurationRetryPeriod:    config.RefreshConfigurationRetryPeriod,
		getObjectsToSynchronizeRetryPeriod: config.GetObjectsToSynchronizeRetryPeriod,
		workers:                            make(map[string]*gitopsWorkerHolder),
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
		ctx, cancel := context.WithCancel(ctx)
		defer cancel() // ensure streaming call is canceled
		req := &agentrpc.ConfigurationRequest{
			CommitId: lastProcessedCommitId,
		}
		res, err := a.kasClient.GetConfiguration(ctx, req)
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				a.log.Warn("GetConfiguration failed", zap.Error(err))
			}
			return
		}
		for {
			config, err := res.Recv()
			if err != nil {
				switch {
				case errors.Is(err, io.EOF):
				case grpctool.RequestCanceled(err):
				default:
					a.log.Warn("GetConfiguration.Recv failed", zap.Error(err))
				}
				return
			}
			lastProcessedCommitId = config.CommitId
			applyDefaultsToConfiguration(config.Configuration)
			err = a.applyConfiguration(config.Configuration)
			if err != nil {
				a.log.Error("Failed to apply configuration", zap.Error(err))
				continue
			}
		}
	}
}

func (a *Agent) applyConfiguration(config *agentcfg.AgentConfiguration) error {
	a.log.Debug("Applying configuration", agentConfig(config))
	err := a.applyObservabilityConfiguration(config.Observability) // Should be called first to configure logging ASAP
	if err != nil {
		return fmt.Errorf("logging: %v", err)
	}
	err = a.applyGitOpsConfiguration(config.Gitops)
	if err != nil {
		return fmt.Errorf("gitops: %v", err)
	}
	return nil
}

func (a *Agent) applyObservabilityConfiguration(obs *agentcfg.ObservabilityCF) error {
	level, err := logz.LevelFromString(obs.Logging.Level.String())
	if err != nil {
		return err
	}
	a.logLevel.SetLevel(level)
	return nil
}

func (a *Agent) applyGitOpsConfiguration(gitops *agentcfg.GitopsCF) error {
	err := a.configureWorkers(gitops.ManifestProjects)
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
			return fmt.Errorf("duplicate project id: %s", project.Id)
		}
		newSetOfProjects.Insert(project.Id)
		workerHolder := a.workers[project.Id]
		if workerHolder == nil { // New project added
			projectsToStartWorkersFor = append(projectsToStartWorkersFor, project)
		} else { // We have a worker for this project already
			if proto.Equal(project, workerHolder.worker.projectConfiguration) {
				// Worker's configuration hasn't changed, nothing to do here
				continue
			}
			a.log.Info("Configuration has been updated, restarting synchronization worker", logz.ProjectId(project.Id))
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
		a.log.Info("Stopping synchronization worker", logz.ProjectId(projectId))
		workerHolder.stop()
		delete(a.workers, projectId)
	}

	// Wait for stopped workers to finish.
	for _, workerHolder := range workersToStop {
		projectId := workerHolder.worker.projectConfiguration.Id
		a.log.Info("Waiting for synchronization worker to stop", logz.ProjectId(projectId))
		workerHolder.wg.Wait()
	}

	// Start new workers for new projects or because of updated configuration.
	for _, project := range projectsToStartWorkersFor {
		a.startNewWorker(project)
	}
	return nil
}

func (a *Agent) startNewWorker(project *agentcfg.ManifestProjectCF) {
	l := a.log.With(logz.ProjectId(project.Id))
	l.Info("Starting synchronization worker")
	worker := &gitopsWorker{
		kasClient:                          a.kasClient,
		engineFactory:                      a.engineFactory,
		getObjectsToSynchronizeRetryPeriod: a.getObjectsToSynchronizeRetryPeriod,
		synchronizerConfig: synchronizerConfig{
			log:                  l,
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

func (f *DefaultGitOpsEngineFactory) New(engineOpts []engine.Option, cacheOpts []cache.UpdateSettingsFunc) engine.GitOpsEngine {
	return engine.NewEngine(
		f.KubeClientConfig,
		cache.NewClusterCache(f.KubeClientConfig, cacheOpts...),
		engineOpts...,
	)
}

func applyDefaultsToConfiguration(config *agentcfg.AgentConfiguration) {
	protodefault.NotNil(&config.Observability)
	protodefault.NotNil(&config.Observability.Logging)
	protodefault.NotNil(&config.Gitops)
}
