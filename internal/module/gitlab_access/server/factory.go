package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitlab_access/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
)

type Factory struct {
}

func (f *Factory) New(config *modserver.Config) modserver.Module {
	sv, err := grpctool.NewStreamVisitor(&rpc.Request{})
	if err != nil {
		// This is a coding error, should never happen.
		panic(err)
	}
	m := &module{
		log:           config.Log,
		api:           config.Api,
		gitLabClient:  config.GitLabClient,
		streamVisitor: sv,
	}
	rpc.RegisterGitlabAccessServer(config.AgentServer, m)
	return m
}
