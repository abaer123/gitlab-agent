package agentk

import "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/modagent"

var (
	_ modagent.API = &api{}
)
