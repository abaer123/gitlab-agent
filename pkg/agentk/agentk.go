package agentk

import (
	"context"
	"io"
	"time"

	"gitlab.com/ash2k/gitlab-agent/pkg/agentrpc"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Agent struct {
	Client agentrpc.GitLabServiceClient
}

func (a *Agent) Run(ctx context.Context) error {
	err := wait.PollImmediateUntil(10*time.Second, a.refreshConfiguration(ctx), ctx.Done())
	if err == wait.ErrWaitTimeout {
		return nil // all good, ctx is done
	}
	return err
}

func (a *Agent) refreshConfiguration(ctx context.Context) wait.ConditionFunc {
	return func() (bool /*done*/, error) {
		req := &agentrpc.ConfigurationRequest{}
		res, err := a.Client.GetConfiguration(ctx, req)
		if err != nil {
			// TODO log
			return false, nil // nil error to keep polling
		}
		for {
			config, err := res.Recv()
			if err != nil {
				if err != io.EOF {
					// TODO log
				}
				return false, nil // nil error to keep polling
			}
			a.applyConfiguration(config)
		}
	}
}

func (a *Agent) applyConfiguration(config *agentrpc.ConfigurationResponse) {
	// TODO
}
