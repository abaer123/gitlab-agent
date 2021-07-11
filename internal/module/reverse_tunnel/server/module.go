package server

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel"
)

type module struct {
}

func (m *module) Run(ctx context.Context) error {
	return nil
}

func (m *module) Name() string {
	return reverse_tunnel.ModuleName
}
