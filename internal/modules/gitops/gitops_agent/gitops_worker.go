package gitops_agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/modclient"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/argoproj/gitops-engine/pkg/engine"
	"github.com/ash2k/stager"
	"github.com/go-logr/zapr"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/retry"
	"go.uber.org/zap"
)

const (
	engineRunRetryPeriod = 10 * time.Second
)

type gitopsWorker struct {
	api                                modclient.AgentAPI
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
	err := retry.PollImmediateUntil(ctx, engineRunRetryPeriod, func() (bool /*done*/, error) {
		var err error
		stopEngine, err = eng.Run()
		if err != nil {
			d.log.Warn("engine.Run() failed", zap.Error(err))
			return false, nil // nil error to keep polling
		}
		return true, nil
	})
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
		res, err := d.api.GetObjectsToSynchronize(ctx, &agentrpc.ObjectsToSynchronizeRequest{
			ProjectId: d.projectConfiguration.Id,
			CommitId:  lastProcessedCommitId,
			Paths:     d.projectConfiguration.Paths,
		})
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				d.log.Warn("GetObjectsToSynchronize failed", zap.Error(err))
			}
			return
		}
		var (
			state            desiredState
			headersReceived  bool
			trailersReceived bool
		)
	objectStream:
		for {
			objectsResp, err := res.Recv()
			if err != nil {
				switch {
				case errors.Is(err, io.EOF):
					break objectStream
				case grpctool.RequestCanceled(err):
				default:
					d.log.Warn("GetObjectsToSynchronize.Recv failed", zap.Error(err))
				}
				return
			}
			switch msg := objectsResp.Message.(type) {
			case *agentrpc.ObjectsToSynchronizeResponse_Headers_:
				headersReceived = true
				state.commitId = msg.Headers.CommitId
			case *agentrpc.ObjectsToSynchronizeResponse_Object_:
				lastIdx := len(state.sources) - 1
				object := msg.Object
				if lastIdx >= 0 && state.sources[lastIdx].name == object.Source {
					// Same source, append to the actual slice
					state.sources[lastIdx].data = append(state.sources[lastIdx].data, object.Data...)
					continue
				}
				state.sources = append(state.sources, stateSource{
					name: object.Source,
					data: object.Data,
				})
			case *agentrpc.ObjectsToSynchronizeResponse_Trailers_:
				trailersReceived = true
			default:
				d.log.Error(fmt.Sprintf("GetObjectsToSynchronize.Recv returned an unexpected type: %T", objectsResp.Message))
				return
			}
		}
		switch {
		case !headersReceived && !trailersReceived:
			if len(state.sources) > 0 {
				// This should never happen.
				s.log.Error(fmt.Sprintf(
					"Server didn't send both headers and trailers for objects stream, but sent some objects. It's a bug! Number of sources received: %d, commit id: %s",
					len(state.sources),
					state.commitId),
				)
				//} else {
				// Normal clean shutdown of a connection that has been open more than connection max age duration.
			}
		case headersReceived && trailersReceived:
			// All good, work on received state
			if s.setDesiredState(ctx, state) {
				lastProcessedCommitId = state.commitId
			}
		default:
			// This should never happen.
			s.log.Error(fmt.Sprintf(
				"Server didn't send both headers (%t) and trailers (%t) for objects stream. It's a bug! Number of sources received: %d, commit id: %s",
				headersReceived,
				trailersReceived,
				len(state.sources),
				state.commitId),
			)
		}
	}
}
