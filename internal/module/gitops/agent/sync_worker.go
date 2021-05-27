package agent

import (
	"bytes"
	"context"
	"os"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
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
	opts     apply.Options
}

type syncWorker struct {
	log     *zap.Logger
	applier Applier
}

func newSyncWorker(log *zap.Logger, applier Applier) *syncWorker {
	return &syncWorker{
		log:     log,
		applier: applier,
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
	events := s.applier.Run(job.ctx, job.invInfo, job.objects, job.opts)
	//The printer will print updates from the channel. It will block
	//until the channel is closed.
	printer := printers.GetPrinter(printers.JSONPrinter, genericclioptions.IOStreams{
		In:     &bytes.Buffer{}, // nothing to read
		Out:    os.Stderr,
		ErrOut: os.Stderr,
	})
	return printer.Print(events, common.DryRunNone)
}
