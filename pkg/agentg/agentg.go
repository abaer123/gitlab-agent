package agentg

import (
	"time"

	"gitlab.com/ash2k/gitlab-agent/pkg/agentrpc"
)

type Agent struct {
	ReloadConfigurationPeriod time.Duration
}

func (a *Agent) GetConfiguration(req *agentrpc.ConfigurationRequest, configStream agentrpc.GitLabService_GetConfigurationServer) error {
	ctx := configStream.Context()
	t := time.NewTicker(a.ReloadConfigurationPeriod)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			err := a.sendConfiguration(configStream)
			if err != nil {
				return err
			}
		}
	}
}

func (a *Agent) sendConfiguration(configStream agentrpc.GitLabService_GetConfigurationServer) error {
	config, err := a.fetchConfiguration()
	if err != nil {
		// TODO log
		return nil // don't want to close the response stream, so report no error
	}
	return configStream.Send(config)
}

func (a *Agent) fetchConfiguration() (*agentrpc.ConfigurationResponse, error) {
	// TODO fetch configuration from per-agent git repo
	return &agentrpc.ConfigurationResponse{}, nil
}
