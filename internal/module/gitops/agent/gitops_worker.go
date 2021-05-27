package agent

import (
	"context"

	"github.com/ash2k/stager"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

type Applier interface {
	Initialize() error
	Run(ctx context.Context, invInfo inventory.InventoryInfo, objects []*unstructured.Unstructured, options apply.Options) <-chan event.Event
}

type ApplierFactory interface {
	New() Applier
}

type GitopsWorkerFactory interface {
	New(int64, *agentcfg.ManifestProjectCF) GitopsWorker
}

type GitopsWorker interface {
	Run(context.Context)
}

type defaultGitopsWorker struct {
	objWatcher     rpc.ObjectsToSynchronizeWatcherInterface
	applierFactory ApplierFactory
	applierBackoff retry.BackoffManagerFactory
	synchronizerConfig
}

func (w *defaultGitopsWorker) Run(ctx context.Context) {
	applier := w.applierFactory.New()
	err := retry.PollWithBackoff(ctx, w.applierBackoff(), true, 0, func() (error, retry.AttemptResult) {
		err := applier.Initialize()
		if err != nil {
			w.log.Error("Applier.Initialize() failed", zap.Error(err))
			return nil, retry.Backoff
		}
		return nil, retry.Done
	})
	if err != nil {
		// context is done
		return
	}
	s := newSynchronizer(w.synchronizerConfig, applier)
	st := stager.New()
	stage := st.NextStage()
	stage.Go(func(ctx context.Context) error {
		s.run(ctx)
		return nil
	})
	stage = st.NextStage()
	stage.Go(func(ctx context.Context) error {
		req := &rpc.ObjectsToSynchronizeRequest{
			ProjectId: w.project.Id,
			Paths:     w.project.Paths,
		}
		w.objWatcher.Watch(ctx, req, func(ctx context.Context, data rpc.ObjectsToSynchronizeData) {
			s.setDesiredState(ctx, data)
		})
		return nil
	})
	_ = st.Run(ctx) // no errors possible
}

type defaultApplierFactory struct {
	provider provider.Provider
}

func (f *defaultApplierFactory) New() Applier {
	return apply.NewApplier(f.provider)
}

type defaultGitopsWorkerFactory struct {
	log                   *zap.Logger
	applierFactory        ApplierFactory
	k8sUtilFactory        util.Factory
	gitopsClient          rpc.GitopsClient
	watchBackoffFactory   retry.BackoffManagerFactory
	applierBackoffFactory retry.BackoffManagerFactory
}

func (f *defaultGitopsWorkerFactory) New(agentId int64, project *agentcfg.ManifestProjectCF) GitopsWorker {
	l := f.log.With(logz.ProjectId(project.Id))
	return &defaultGitopsWorker{
		objWatcher: &rpc.ObjectsToSynchronizeWatcher{
			Log:          l,
			GitopsClient: f.gitopsClient,
			Backoff:      f.watchBackoffFactory,
		},
		applierFactory: f.applierFactory,
		applierBackoff: f.applierBackoffFactory,
		synchronizerConfig: synchronizerConfig{
			log:            l,
			agentId:        agentId,
			project:        project,
			k8sUtilFactory: f.k8sUtilFactory,
		},
	}
}
