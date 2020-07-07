package agentk

import (
	"context"
	"io"
	"time"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	engineRunRetryPeriod               = 10 * time.Second
	getObjectsToSynchronizeRetryPeriod = 10 * time.Second
)

type deploymentWorker struct {
	engineFactory GitOpsEngineFactory
	synchronizerConfig
}

func (d *deploymentWorker) Run(ctx context.Context) {
	eng := d.engineFactory.New(cache.SetPopulateResourceInfoHandler(populateResourceInfoHandler))
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
	defer stopEngine.Close()

	s := synchronizer{
		ctx:                ctx,
		eng:                eng,
		synchronizerConfig: d.synchronizerConfig,
	}

	_ = wait.PollImmediateUntil(getObjectsToSynchronizeRetryPeriod, func() (bool /*done*/, error) {
		s.run()
		return false, nil // never done, never error. Polling is interrupted by ctx
	}, ctx.Done())
}
