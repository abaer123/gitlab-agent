package server

import (
	"context"
	"net"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/observability"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"go.uber.org/zap"
)

type module struct {
	log        *zap.Logger
	cfg        *kascfg.ObservabilityCF
	gatherer   prometheus.Gatherer
	registerer prometheus.Registerer
	serverName string
}

func (m *module) Run(ctx context.Context) (retErr error) {
	lis, err := net.Listen(m.cfg.Listen.Network.String(), m.cfg.Listen.Address)
	if err != nil {
		return err
	}
	defer errz.SafeClose(lis, &retErr)

	m.log.Info("Observability endpoint is up",
		logz.NetNetworkFromAddr(lis.Addr()),
		logz.NetAddressFromAddr(lis.Addr()),
	)

	metricSrv := metricServer{
		Name:                  m.serverName,
		Listener:              lis,
		PrometheusUrlPath:     m.cfg.Prometheus.UrlPath,
		LivenessProbeUrlPath:  m.cfg.LivenessProbe.UrlPath,
		ReadinessProbeUrlPath: m.cfg.ReadinessProbe.UrlPath,
		Gatherer:              m.gatherer,
		Registerer:            m.registerer,
	}
	return metricSrv.Run(ctx)
}

func (m *module) Name() string {
	return observability.ModuleName
}
