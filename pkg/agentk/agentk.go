package agentk

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/kubernetes-management-ng/gitlab-agent/pkg/agentrpc"
	"gitlab.com/gitlab-org/labkit/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
			log.WithError(err).Warn("GetConfiguration failed")
			return false, nil // nil error to keep polling
		}
		for {
			config, err := res.Recv()
			if err != nil {
				switch {
				case err == io.EOF:
				case status.Code(err) == codes.DeadlineExceeded:
				case status.Code(err) == codes.Canceled:
				default:
					log.WithError(err).Warn("GetConfiguration.Recv failed")
				}
				return false, nil // nil error to keep polling
			}
			a.applyConfiguration(config.Configuration)
		}
	}
}

func (a *Agent) applyConfiguration(config *agentrpc.AgentConfiguration) {
	fmt.Fprintf(os.Stderr, "%v\n", config)

}
