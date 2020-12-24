package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
)

var (
	_ modserver.Module       = &module{}
	_ modserver.Factory      = &Factory{}
	_ rpc.AgentTrackerServer = &module{}
)
