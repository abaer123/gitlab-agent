package agentk

import (
	"context"
	"fmt"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/argoproj/gitops-engine/pkg/engine"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/labkit/log"
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
			s.log.WithError(err).Warn("Synchronization failed")
		}
	}
}

func (s *syncWorker) synchronize(job syncJob) error {
	result, err := s.engine.Sync(job.ctx, job.objects, s.isManaged, job.commitId, "" /*TODO namespace*/)
	if err != nil {
		// TODO check ctx.Err() https://github.com/argoproj/gitops-engine/pull/140
		return fmt.Errorf("engine.Sync failed: %v", err)
	}
	for _, res := range result {
		s.log.WithFields(log.Fields{
			api.ResourceKey: res.ResourceKey.String(),
			api.SyncResult:  res.Message,
		}).Info("Synced")
	}
	return nil
}

func (s *syncWorker) isManaged(r *cache.Resource) bool {
	return r.Info.(*resourceInfo).gcMark == "managed" // TODO
}
