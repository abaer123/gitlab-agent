package observability_agent

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/modagent"
)

var (
	_ modagent.Module  = &module{}
	_ modagent.Factory = &Factory{}
)
