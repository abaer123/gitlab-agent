package agentk

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/argoproj/gitops-engine/pkg/engine"
	"github.com/ash2k/stager"
	"github.com/go-logr/zapr"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/grpctools"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/retry"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	engineRunRetryPeriod = 10 * time.Second
)

type gitopsWorker struct {
	kasClient                          agentrpc.KasClient
	engineFactory                      GitOpsEngineFactory
	getObjectsToSynchronizeRetryPeriod time.Duration
	synchronizerConfig
}

func (d *gitopsWorker) Run(ctx context.Context) {
	l := zapr.NewLogger(d.log)
	eng := d.engineFactory.New(
		[]engine.Option{
			engine.WithLogr(l),
		},
		[]cache.UpdateSettingsFunc{
			cache.SetPopulateResourceInfoHandler(populateResourceInfoHandler),
			cache.SetSettings(cache.Settings{
				ResourcesFilter: resourcesFilter{
					resourceInclusions: d.synchronizerConfig.projectConfiguration.ResourceInclusions,
					resourceExclusions: d.synchronizerConfig.projectConfiguration.ResourceExclusions,
				},
			}),
			cache.SetLogr(l),
		},
	)
	var stopEngine engine.StopFunc
	err := wait.PollImmediateUntil(engineRunRetryPeriod, func() (bool /*done*/, error) {
		var err error
		stopEngine, err = eng.Run()
		if err != nil {
			d.log.Warn("engine.Run() failed", zap.Error(err))
			return false, nil // nil error to keep polling
		}
		return true, nil
	}, ctx.Done())
	if err != nil {
		// context is done
		return
	}
	defer stopEngine()
	s := newSynchronizer(d.synchronizerConfig, eng)
	st := stager.New()
	stage := st.NextStage()
	stage.Go(func(ctx context.Context) error {
		s.run(ctx)
		return nil
	})
	stage = st.NextStage()
	stage.Go(func(ctx context.Context) error {
		retry.JitterUntil(ctx, d.getObjectsToSynchronizeRetryPeriod, d.getObjectsToSynchronize(s))
		return nil
	})
	_ = st.Run(ctx) // no errors possible
}

func (d *gitopsWorker) getObjectsToSynchronize(s *synchronizer) func(context.Context) {
	var lastProcessedCommitId string
	return func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel() // ensure streaming call is canceled
		req := &agentrpc.ObjectsToSynchronizeRequest{
			ProjectId: d.projectConfiguration.Id,
			CommitId:  lastProcessedCommitId,
			Paths:     d.projectConfiguration.Paths,
		}
		res, err := d.kasClient.GetObjectsToSynchronize(ctx, req)
		if err != nil {
			if !grpctools.RequestCanceled(err) {
				d.log.Warn("GetObjectsToSynchronize failed", zap.Error(err))
			}
			return
		}
		for {
			objectsResp, err := res.Recv()
			if err != nil {
				switch {
				case errors.Is(err, io.EOF):
				case grpctools.RequestCanceled(err):
				default:
					d.log.Warn("GetObjectsToSynchronize.Recv failed", zap.Error(err))
				}
				return
			}
			s.setDesiredState(ctx, objectsResp)
			lastProcessedCommitId = objectsResp.CommitId
		}
	}
}
