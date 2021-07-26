package agent

import (
	"bytes"
	"context"
	"os"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/cmd/printers"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
)

type syncJob struct {
	ctx      context.Context
	commitId string
	invInfo  inventory.InventoryInfo
	objects  []*unstructured.Unstructured
}

type syncWorker struct {
	log             *zap.Logger
	reapplyInterval time.Duration
	applier         Applier
	applierBackoff  retry.BackoffManager
	applyOptions    apply.Options
}

func (s *syncWorker) Run(jobs <-chan syncJob) {
	for job := range jobs {
		l := s.log.With(logz.CommitId(job.commitId))
		_ = retry.PollWithBackoff(job.ctx, s.applierBackoff, true, s.reapplyInterval, func() (error, retry.AttemptResult) {
			l.Info("Synchronizing objects")
			err := s.synchronize(job)
			if err != nil {
				if errz.ContextDone(err) {
					l.Info("Synchronization was canceled", logz.Error(err))
				} else {
					l.Warn("Synchronization failed", logz.Error(err))
				}
				return nil, retry.Backoff
			}
			l.Info("Objects synchronized")
			return nil, retry.Continue
		})
	}
}

func (s *syncWorker) synchronize(job syncJob) error {
	events := s.applier.Run(job.ctx, job.invInfo, job.objects, s.applyOptions)
	//The printer will print updates from the channel. It will block
	//until the channel is closed.
	printer := printers.GetPrinter(printers.JSONPrinter, genericclioptions.IOStreams{
		In:     &bytes.Buffer{}, // nothing to read
		Out:    os.Stderr,
		ErrOut: os.Stderr,
	})
	return printer.Print(events, common.DryRunNone, true)
}
