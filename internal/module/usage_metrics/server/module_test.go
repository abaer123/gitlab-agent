package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	_ modserver.Module        = &module{}
	_ modserver.Factory       = &Factory{}
	_ modserver.ApplyDefaults = ApplyDefaults
)

func TestSendUsage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, tracker, client, _ := setupModule(t)
	counters := map[string]int64{
		"x": 5,
	}
	ud := &usage_metrics.UsageData{Counters: counters}
	gomock.InOrder(
		tracker.EXPECT().
			CloneUsageData().
			Return(ud, false),
		client.EXPECT().
			SendUsage(gomock.Any(), gitlab.UsageData(counters)),
		tracker.EXPECT().
			Subtract(ud),
		tracker.EXPECT().
			CloneUsageData().
			DoAndReturn(func() (*usage_metrics.UsageData, bool) {
				cancel()
				return &usage_metrics.UsageData{}, true
			}),
	)
	require.NoError(t, m.Run(ctx))
}

func TestSendUsageFailureAndRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, tracker, client, mockApi := setupModule(t)
	expectedErr := errors.New("expected error")
	counters1 := map[string]int64{
		"x": 5,
	}
	ud1 := &usage_metrics.UsageData{Counters: counters1}
	counters2 := map[string]int64{
		"x": 6,
	}
	ud2 := &usage_metrics.UsageData{Counters: counters2}
	gomock.InOrder(
		tracker.EXPECT().
			CloneUsageData().
			Return(ud1, false),
		client.EXPECT().
			SendUsage(gomock.Any(), gitlab.UsageData(counters1)).
			Return(expectedErr),
		mockApi.EXPECT().LogAndCapture(gomock.Any(), gomock.Any(), "Failed to send usage data", expectedErr).
			DoAndReturn(func(ctx context.Context, log *zap.Logger, msg string, err error) {
				cancel()
			}),
		tracker.EXPECT().
			CloneUsageData().
			Return(ud2, false),
		client.EXPECT().
			SendUsage(gomock.Any(), gitlab.UsageData(counters2)),
		tracker.EXPECT().
			Subtract(ud2),
		tracker.EXPECT().
			CloneUsageData().
			DoAndReturn(func() (*usage_metrics.UsageData, bool) {
				cancel()
				return &usage_metrics.UsageData{}, true
			}),
	)
	require.NoError(t, m.Run(ctx))
}

func setupModule(t *testing.T) (modserver.Module, *mock_usage_metrics.MockUsageTrackerInterface, *mock_gitlab.MockClientInterface, *mock_modserver.MockAPI) {
	ctrl := gomock.NewController(t)
	tracker := mock_usage_metrics.NewMockUsageTrackerInterface(ctrl)
	client := mock_gitlab.NewMockClientInterface(ctrl)
	mockApi := mock_modserver.NewMockAPI(ctrl)
	f := Factory{
		UsageTracker: tracker,
		GitLabClient: client,
	}
	config := &kascfg.ConfigurationFile{}
	ApplyDefaults(config)
	config.Observability.UsageReportingPeriod = durationpb.New(10 * time.Millisecond)
	m := f.New(&modserver.Config{
		Log:    zaptest.NewLogger(t),
		Api:    mockApi,
		Config: config,
	})
	return m, tracker, client, mockApi
}
