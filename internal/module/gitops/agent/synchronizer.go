package agent

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
)

// synchronizerConfig holds configuration for a synchronizer.
type synchronizerConfig struct {
	log            *zap.Logger
	agentId        int64
	project        *agentcfg.ManifestProjectCF
	k8sUtilFactory util.Factory
}

type synchronizer struct {
	synchronizerConfig
	applier      Applier
	desiredState chan rpc.ObjectsToSynchronizeData
}

func newSynchronizer(config synchronizerConfig, applier Applier) *synchronizer {
	return &synchronizer{
		synchronizerConfig: config,
		applier:            applier,
		desiredState:       make(chan rpc.ObjectsToSynchronizeData),
	}
}

func (s *synchronizer) setDesiredState(ctx context.Context, state rpc.ObjectsToSynchronizeData) bool {
	select {
	case <-ctx.Done():
		return false
	case s.desiredState <- state:
		return true
	}
}

func (s *synchronizer) run(ctx context.Context) {
	jobs := make(chan syncJob)
	sw := newSyncWorker(s.log, s.applier)
	var wg wait.Group
	defer wg.Wait()   // Wait for sw to exit
	defer close(jobs) // Close jobs to signal sw there is no more work to be done
	wg.Start(func() {
		sw.run(jobs) // Start sw
	})

	var (
		jobsCh    chan syncJob
		newJob    syncJob
		jobCancel context.CancelFunc
	)
	defer func() {
		if jobCancel != nil {
			jobCancel()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return // nolint: govet
		case state := <-s.desiredState:
			objs, err := s.decodeObjectsToSynchronize(state.Sources)
			if err != nil {
				s.log.Warn("Failed to decode GitOps objects", zap.Error(err), logz.CommitId(state.CommitId))
				continue
			}
			if jobCancel != nil {
				jobCancel() // Cancel running/pending job ASAP
			}
			newJob = syncJob{
				commitId: state.CommitId,
				invInfo:  s.inventoryInfo(),
				objects:  objs,
				opts: apply.Options{
					ServerSideOptions: common.ServerSideOptions{
						ServerSideApply: true,
						ForceConflicts:  false, // want to fail on conflicts - just out of caution TODO make configurable?
						FieldManager:    "agentk",
					},
					ReconcileTimeout:       time.Hour, // TODO make configurable?
					PollInterval:           0,         // use default value
					EmitStatusEvents:       true,
					NoPrune:                false,
					DryRunStrategy:         common.DryRunNone,
					PrunePropagationPolicy: metav1.DeletePropagationBackground, // TODO make configurable?
					PruneTimeout:           time.Hour,                          // TODO make configurable?
					InventoryPolicy:        inventory.InventoryPolicyMustMatch, // TODO make configurable
				},
			}
			newJob.ctx, jobCancel = context.WithCancel(context.Background()) // nolint: govet
			jobsCh = jobs                                                    // Enable select case
		case jobsCh <- newJob: // Try to send new job to syncWorker. This case is active only when jobsCh is not nil
			// Success!
			newJob = syncJob{} // Erase contents to help GC
			jobsCh = nil       // Disable this select case (send to nil channel blocks forever)
		}
	}
}

func (s *synchronizer) decodeObjectsToSynchronize(sources []rpc.ObjectSource) ([]*unstructured.Unstructured, error) {
	if len(sources) == 0 {
		return nil, nil
	}
	// TODO allow enforcing namespace
	builder := s.k8sUtilFactory.NewBuilder().
		//ContinueOnError(). // TODO collect errors and report them all
		Flatten().
		NamespaceParam(s.project.DefaultNamespace).
		DefaultNamespace().
		Unstructured()
	for _, source := range sources {
		builder.Stream(bytes.NewReader(source.Data), source.Name)
	}
	var res []*unstructured.Unstructured
	err := builder.Do().Visit(func(info *resource.Info, err error) error {
		if err != nil {
			// TODO collect errors and report them all
			return err
		}
		un := info.Object.(*unstructured.Unstructured)
		res = append(res, un)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}
func (s *synchronizer) inventoryInfo() inventory.InventoryInfo {
	id := inventoryId(s.agentId, s.project.Id)
	return inventory.WrapInventoryInfoObj(&unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "inventory-" + id,
				"namespace": s.project.DefaultNamespace,
				"labels": map[string]interface{}{
					common.InventoryLabel: id,
				},
			},
		},
	})
}

func inventoryId(agentId int64, projectId string) string {
	h := fnv.New128()
	_, _ = h.Write([]byte(projectId))
	return fmt.Sprintf("%d-%x", agentId, h.Sum(nil))
}
