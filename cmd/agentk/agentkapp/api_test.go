package agentkapp

import "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"

var (
	_ modagent.API = &agentAPI{}
)