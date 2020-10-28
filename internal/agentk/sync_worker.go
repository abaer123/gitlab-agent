package agentk

import (
	"context"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/argoproj/gitops-engine/pkg/engine"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/errz"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type syncJob struct {
	ctx      context.Context
	commitId string
	objects  []*unstructured.Unstructured
}

type syncWorker struct {
	synchronizerConfig
	engine engine.GitOpsEngine
}

func newSyncWorker(config synchronizerConfig, engine engine.GitOpsEngine) *syncWorker {
	return &syncWorker{
		synchronizerConfig: config,
		engine:             engine,
	}
}

func (s *syncWorker) run(jobs <-chan syncJob) {
	for job := range jobs {
		err := s.synchronize(job)
		if err != nil {
			if errz.ContextDone(err) {
				s.log.Info("Synchronization was canceled", zap.Error(err))
			} else {
				s.log.Warn("Synchronization failed", zap.Error(err))
			}
		}
	}
}

func (s *syncWorker) synchronize(job syncJob) error {
	result, err := s.engine.Sync(job.ctx, job.objects, s.isManaged, job.commitId, s.projectConfiguration.DefaultNamespace)
	if err != nil {
		return err // don't wrap
	}
	for _, res := range result {
		s.log.Info("Synced", engineResourceKey(res.ResourceKey), engineSyncResult(res.Message))
	}
	return nil
}

func (s *syncWorker) isManaged(r *cache.Resource) bool {
	return r.Info.(*resourceInfo).gcMark == "managed" // TODO
}
