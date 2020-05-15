package agentk

import (
	"context"
	"log"

	"gitlab.com/ash2k/gitlab-agent/pkg/agentrpc"
)

type Agent struct {
	Client agentrpc.GitLabServiceClient
}

func (a *Agent) Run(ctx context.Context) error {
	req := &agentrpc.ConfigurationRequest{}
	res, err := a.Client.GetConfiguration(ctx, req)
	if err != nil {
		// TODO maybe retry?
		return err
	}
	log.Printf("Fetched configuration: %T", res)
	<-ctx.Done()
	return nil
}
