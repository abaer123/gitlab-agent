package server

import (
	"context"
	"net"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/kubernetes_api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"go.uber.org/zap"
)

type module struct {
	log      *zap.Logger
	proxy    kubernetesApiProxy
	listener func() (net.Listener, error)
}

func (m *module) Run(ctx context.Context) error {
	lis, err := m.listener()
	if err != nil {
		return err
	}
	// Error is ignored because kubernetesApiProxy.Run() closes the listener and
	// a second close always produces an error.
	defer lis.Close() // nolint:errcheck

	m.log.Info("Kubernetes API endpoint is up",
		logz.NetNetworkFromAddr(lis.Addr()),
		logz.NetAddressFromAddr(lis.Addr()),
	)
	return m.proxy.Run(ctx, lis)
}

func (m *module) Name() string {
	return kubernetes_api.ModuleName
}

type nopModule struct {
}

func (m nopModule) Run(ctx context.Context) error {
	return nil
}

func (m nopModule) Name() string {
	return kubernetes_api.ModuleName
}
