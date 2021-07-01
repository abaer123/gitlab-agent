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
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
	Log       *zap.Logger
	AgentMeta *modshared.AgentMeta
	Client    AgentConfigurationClient
	Backoff   retry.BackoffManagerFactory
}

func (w *ConfigurationWatcher) Watch(ctx context.Context, callback ConfigurationCallback) {
	var lastProcessedCommitId string
	_ = retry.PollWithBackoff(ctx, w.Backoff(), true, 0 /* doesn't matter */, func() (error, retry.AttemptResult) {
		ctx, cancel := context.WithCancel(ctx) // nolint:govet
		defer cancel()                         // ensure streaming call is canceled
		var responseMD metadata.MD
		res, err := w.Client.GetConfiguration(ctx, &ConfigurationRequest{
			CommitId:  lastProcessedCommitId,
			AgentMeta: w.AgentMeta,
		}, grpc.Header(&responseMD))
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				w.Log.Warn("GetConfiguration failed", logz.Error(grpctool.MaybeWrapWithCorrelationId(err, responseMD)))
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
					w.Log.Warn("GetConfiguration.Recv failed", logz.Error(grpctool.MaybeWrapWithCorrelationId(err, responseMD)))
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
