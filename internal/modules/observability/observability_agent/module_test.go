package observability_agent

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/modclient"
)

var (
	_ modclient.Module  = &Module{}
	_ modclient.Factory = &Factory{}
)
