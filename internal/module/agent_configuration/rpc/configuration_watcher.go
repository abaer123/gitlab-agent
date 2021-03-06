package rpc

import (
	"context"
	"errors"
	"io"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modshared"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
)

type ConfigurationData struct {
	CommitId string
	Config   *agentcfg.AgentConfiguration
}

type ConfigurationCallback func(context.Context, ConfigurationData)

// ConfigurationWatcherInterface abstracts ConfigurationWatcher.
type ConfigurationWatcherInterface interface {
	Watch(ctx context.Context, callback ConfigurationCallback)
}

type ConfigurationWatcher struct {
	Log         *zap.Logger
	AgentMeta   *modshared.AgentMeta
	Client      AgentConfigurationClient
	RetryPeriod time.Duration
}

func (w *ConfigurationWatcher) Watch(ctx context.Context, callback ConfigurationCallback) {
	var lastProcessedCommitId string
	retry.JitterUntil(ctx, w.RetryPeriod, func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel() // ensure streaming call is canceled
		req := &ConfigurationRequest{
			CommitId:  lastProcessedCommitId,
			AgentMeta: w.AgentMeta,
		}
		res, err := w.Client.GetConfiguration(ctx, req)
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				w.Log.Warn("GetConfiguration failed", zap.Error(err))
			}
			return
		}
		for {
			config, err := res.Recv()
			if err != nil {
				switch {
				case errors.Is(err, io.EOF):
				case grpctool.RequestCanceled(err):
				default:
					w.Log.Warn("GetConfiguration.Recv failed", zap.Error(err))
				}
				return
			}
			callback(ctx, ConfigurationData{
				CommitId: config.CommitId,
				Config:   config.Configuration,
			})
			lastProcessedCommitId = config.CommitId
		}
	})
}
