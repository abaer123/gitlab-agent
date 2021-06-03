package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/configuration_project"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/configuration_project/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
)

type Factory struct {
}

func (f *Factory) New(config *modserver.Config) (modserver.Module, error) {
	rpc.RegisterConfigurationProjectServer(config.ApiServer, &server{
		api:    config.Api,
		gitaly: config.Gitaly,
	})
	return &module{}, nil
}

func (f *Factory) Name() string {
	return configuration_project.ModuleName
}
