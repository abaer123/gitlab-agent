package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitlab_access"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitlab_access/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
)

type Factory struct {
}

func (f *Factory) New(config *modserver.Config) (modserver.Module, error) {
	m := newModule(config.Api, config.GitLabClient)
	rpc.RegisterGitlabAccessServer(config.AgentServer, m)
	return m, nil
}

func (f *Factory) Name() string {
	return gitlab_access.ModuleName
}
