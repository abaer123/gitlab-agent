package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
)

var (
	_ modserver.Module        = &module{}
	_ modserver.Factory       = &Factory{}
	_ modserver.ApplyDefaults = ApplyDefaults
)
