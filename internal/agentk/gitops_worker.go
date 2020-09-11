package agentk

import (
	"context"
	"io"
	"time"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/ash2k/stager"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/retry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	eng := d.engineFactory.New(cache.SetPopulateResourceInfoHandler(populateResourceInfoHandler), cache.SetSettings(cache.Settings{
		ResourcesFilter: resourcesFilter{
			resourceInclusions: d.synchronizerConfig.projectConfiguration.ResourceInclusions,
			resourceExclusions: d.synchronizerConfig.projectConfiguration.ResourceExclusions,
		},
	}))
	var stopEngine io.Closer
	err := wait.PollImmediateUntil(engineRunRetryPeriod, func() (bool /*done*/, error) {
		var err error
		stopEngine, err = eng.Run()
		if err != nil {
			d.log.WithError(err).Warn("engine.Run() failed")
			return false, nil // nil error to keep polling
		}
		return true, nil
	}, ctx.Done())
	if err != nil {
		// context is done
		return
	}
	defer stopEngine.Close() // nolint: errcheck
	st := stager.New()
	defer st.Shutdown()
	stage := st.NextStage()
	s := newSynchronizer(d.synchronizerConfig, eng)
	stage.StartWithContext(s.run)

	retry.JitterUntil(ctx, d.getObjectsToSynchronizeRetryPeriod, d.getObjectsToSynchronize(s))
}

func (d *gitopsWorker) getObjectsToSynchronize(s *synchronizer) func(context.Context) {
	var lastProcessedCommitId string
	return func(ctx context.Context) {
		req := &agentrpc.ObjectsToSynchronizeRequest{
			ProjectId: d.projectConfiguration.Id,
			CommitId:  lastProcessedCommitId,
		}
		res, err := d.kasClient.GetObjectsToSynchronize(ctx, req)
		if err != nil {
			d.log.WithError(err).Warn("GetObjectsToSynchronize failed")
			return
		}
		for {
			objectsResp, err := res.Recv()
			if err != nil {
				switch {
				case err == io.EOF:
				case status.Code(err) == codes.DeadlineExceeded:
				case status.Code(err) == codes.Canceled:
				default:
					d.log.WithError(err).Warn("GetObjectsToSynchronize.Recv failed")
				}
				return
			}
			s.setDesiredState(ctx, objectsResp)
			lastProcessedCommitId = objectsResp.CommitId
		}
	}
}
