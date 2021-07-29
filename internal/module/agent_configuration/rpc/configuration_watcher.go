package rpc

import (
	"context"
	"errors"
	"io"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modshared"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"go.uber.org/zap"
)

type ConfigurationData struct {
	CommitId string
	Config   *agentcfg.AgentConfiguration
}

type ConfigurationCallback func(context.Context, ConfigurationData)

// ConfigurationWatcherInterface abstracts ConfigurationWatcher.
type ConfigurationWatcherInterface interface {
	Watch(context.Context, ConfigurationCallback)
}

type ConfigurationWatcher struct {
	Log        *zap.Logger
	AgentMeta  *modshared.AgentMeta
	Client     AgentConfigurationClient
	PollConfig retry.PollConfigFactory
}

func (w *ConfigurationWatcher) Watch(ctx context.Context, callback ConfigurationCallback) {
	var lastProcessedCommitId string
	_ = retry.PollWithBackoff(ctx, w.PollConfig(), func() (error, retry.AttemptResult) {
		ctx, cancel := context.WithCancel(ctx) // nolint:govet
		defer cancel()                         // ensure streaming call is canceled
		res, err := w.Client.GetConfiguration(ctx, &ConfigurationRequest{
			CommitId:  lastProcessedCommitId,
			AgentMeta: w.AgentMeta,
		})
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				w.Log.Warn("GetConfiguration failed", logz.Error(err))
			}
			return nil, retry.Backoff
		}
		for {
			config, err := res.Recv()
			if err != nil {
				switch {
				case errors.Is(err, io.EOF):
					return nil, retry.ContinueImmediately // immediately reconnect after a clean close
				case grpctool.RequestCanceled(err):
				default:
					w.Log.Warn("GetConfiguration.Recv failed", logz.Error(grpctool.MaybeWrapWithCorrelationId(err, res)))
				}
				return nil, retry.Backoff
			}
			callback(ctx, ConfigurationData{
				CommitId: config.CommitId,
				Config:   config.Configuration,
			})
			lastProcessedCommitId = config.CommitId
		}
	})
}
