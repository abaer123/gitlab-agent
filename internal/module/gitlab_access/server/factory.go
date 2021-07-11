package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitlab_access"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitlab_access/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
)

type Factory struct {
}

func (f *Factory) New(config *modserver.Config) (modserver.Module, error) {
	s := newServer(config.Api, config.GitLabClient)
	rpc.RegisterGitlabAccessServer(config.AgentServer, s)
	return &module{}, nil
}

func (f *Factory) Name() string {
	return gitlab_access.ModuleName
}
