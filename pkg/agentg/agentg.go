package agentg

import (
	"context"

	"gitlab.com/ash2k/gitlab-agent/pkg/agentrpc"
)

type Agent struct {
}

func (a *Agent) GetConfiguraiton(context.Context, *agentrpc.ConfigurationRequest) (*agentrpc.ConfigurationResponse, error) {
	return &agentrpc.ConfigurationResponse{}, nil
}
