package agent

import (
	"bytes"
	"context"
	"fmt"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"go.uber.org/zap"
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
	applyOptions   apply.Options
}

type synchronizer struct {
	synchronizerConfig
	applier Applier
}

func newSynchronizer(config synchronizerConfig, applier Applier) *synchronizer {
	return &synchronizer{
		synchronizerConfig: config,
		applier:            applier,
	}
}

func (s *synchronizer) run(desiredState <-chan rpc.ObjectsToSynchronizeData) {
	jobs := make(chan syncJob)
	sw := newSyncWorker(s.log, s.applier, s.applyOptions)
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
		case state, ok := <-desiredState:
			if !ok {
				return // nolint: govet
			}
			objs, err := s.decodeObjectsToSynchronize(state.Sources)
			if err != nil {
				s.log.Error("Failed to decode GitOps objects", zap.Error(err), logz.CommitId(state.CommitId))
				continue
			}
			invObj, objs, err := s.splitObjects(state.ProjectId, objs)
			if err != nil {
				s.log.Error("Failed to locate inventory object in GitOps objects", zap.Error(err), logz.CommitId(state.CommitId))
				continue
			}
			if jobCancel != nil {
				jobCancel() // Cancel running/pending job ASAP
			}
			newJob = syncJob{
				commitId: state.CommitId,
				invInfo:  inventory.WrapInventoryInfoObj(invObj),
				objects:  objs,
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

func (s *synchronizer) splitObjects(projectId int64, objs []*unstructured.Unstructured) (*unstructured.Unstructured, []*unstructured.Unstructured, error) {
	invs := make([]*unstructured.Unstructured, 0, 1)
	resources := make([]*unstructured.Unstructured, 0, len(objs))
	for _, obj := range objs {
		if inventory.IsInventoryObject(obj) {
			invs = append(invs, obj)
		} else {
			resources = append(resources, obj)
		}
	}
	switch len(invs) {
	case 0:
		return s.defaultInventoryObjTemplate(projectId), resources, nil
	case 1:
		return invs[0], resources, nil
	default:
		return nil, nil, fmt.Errorf("expecting zero or one inventory object, found %d", len(invs))
	}
}

func (s *synchronizer) defaultInventoryObjTemplate(projectId int64) *unstructured.Unstructured {
	id := inventoryId(s.agentId, projectId)
	return &unstructured.Unstructured{
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
	}
}

func inventoryId(agentId, projectId int64) string {
	return fmt.Sprintf("%d-%d", agentId, projectId)
}
