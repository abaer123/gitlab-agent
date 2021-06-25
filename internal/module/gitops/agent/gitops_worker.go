package agent

import (
	"context"

	"github.com/ash2k/stager"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

const (
	dryRunStrategyNone   = "none"
	dryRunStrategyClient = "client"
	dryRunStrategyServer = "server"

	prunePropagationPolicyOrphan     = "orphan"
	prunePropagationPolicyBackground = "background"
	prunePropagationPolicyForeground = "foreground"

	inventoryPolicyMustMatch          = "must_match"
	inventoryPolicyAdoptIfNoInventory = "adopt_if_no_inventory"
	inventoryPolicyAdoptAll           = "adopt_all"
)

var (
	dryRunStrategyMapping = map[string]common.DryRunStrategy{
		dryRunStrategyNone:   common.DryRunNone,
		dryRunStrategyClient: common.DryRunClient,
		dryRunStrategyServer: common.DryRunServer,
	}
	prunePropagationPolicyMapping = map[string]metav1.DeletionPropagation{
		prunePropagationPolicyOrphan:     metav1.DeletePropagationOrphan,
		prunePropagationPolicyBackground: metav1.DeletePropagationBackground,
		prunePropagationPolicyForeground: metav1.DeletePropagationForeground,
	}
	inventoryPolicyMapping = map[string]inventory.InventoryPolicy{
		inventoryPolicyMustMatch:          inventory.InventoryPolicyMustMatch,
		inventoryPolicyAdoptIfNoInventory: inventory.AdoptIfNoInventory,
		inventoryPolicyAdoptAll:           inventory.AdoptAll,
	}
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
	desiredState := make(chan rpc.ObjectsToSynchronizeData)
	st := stager.New()
	stage := st.NextStage()
	stage.Go(func(ctx context.Context) error {
		s := newSynchronizer(w.synchronizerConfig, applier)
		s.run(desiredState)
		return nil
	})
	stage = st.NextStage()
	stage.Go(func(ctx context.Context) error {
		defer close(desiredState)
		req := &rpc.ObjectsToSynchronizeRequest{
			ProjectId: w.project.Id,
			Paths:     w.project.Paths,
		}
		w.objWatcher.Watch(ctx, req, func(ctx context.Context, data rpc.ObjectsToSynchronizeData) {
			select {
			case <-ctx.Done():
			case desiredState <- data:
			}
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
			applyOptions: apply.Options{
				ServerSideOptions: common.ServerSideOptions{
					// It's supported since Kubernetes 1.16, so there should be no reason not to use it.
					// https://kubernetes.io/docs/reference/using-api/server-side-apply/
					ServerSideApply: true,
					// GitOps repository is the source of truth and that's what we are applying, so overwrite any conflicts.
					// https://kubernetes.io/docs/reference/using-api/server-side-apply/#conflicts
					ForceConflicts: true,
					// https://kubernetes.io/docs/reference/using-api/server-side-apply/#field-management
					FieldManager: "agentk",
				},
				ReconcileTimeout:       project.ReconcileTimeout.AsDuration(),
				PollInterval:           0, // use default value
				EmitStatusEvents:       true,
				NoPrune:                !project.GetPrune(),
				DryRunStrategy:         f.mapDryRunStrategy(project.DryRunStrategy),
				PrunePropagationPolicy: f.mapPrunePropagationPolicy(project.PrunePropagationPolicy),
				PruneTimeout:           project.PruneTimeout.AsDuration(),
				InventoryPolicy:        f.mapInventoryPolicy(project.InventoryPolicy),
			},
		},
	}
}

func (f *defaultGitopsWorkerFactory) mapDryRunStrategy(strategy string) common.DryRunStrategy {
	ret, ok := dryRunStrategyMapping[strategy]
	if !ok {
		// This shouldn't happen because we've checked the value in DefaultAndValidateConfiguration().
		// Just being extra cautious.
		f.log.Sugar().Errorf("Invalid dry-run strategy: %q, using client dry-run for safety - NO CHANGES WILL BE APPLIED!", strategy)
		ret = common.DryRunClient
	}
	return ret
}

func (f *defaultGitopsWorkerFactory) mapPrunePropagationPolicy(policy string) metav1.DeletionPropagation {
	ret, ok := prunePropagationPolicyMapping[policy]
	if !ok {
		// This shouldn't happen because we've checked the value in DefaultAndValidateConfiguration().
		// Just being extra cautious.
		f.log.Sugar().Errorf("Invalid prune propagation policy: %q, defaulting to %s", policy, metav1.DeletePropagationForeground)
		ret = metav1.DeletePropagationForeground
	}
	return ret
}

func (f *defaultGitopsWorkerFactory) mapInventoryPolicy(policy string) inventory.InventoryPolicy {
	ret, ok := inventoryPolicyMapping[policy]
	if !ok {
		// This shouldn't happen because we've checked the value in DefaultAndValidateConfiguration().
		// Just being extra cautious.
		f.log.Sugar().Errorf("Invalid inventory policy: %q, defaulting to 'must match'", policy)
		ret = inventory.InventoryPolicyMustMatch
	}
	return ret
}
