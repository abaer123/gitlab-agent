package kas

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/matcher"
	"gitlab.com/gitlab-org/labkit/errortracking"
)

func TestSendUsage(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k, mockCtrl, _, gitlabClient, _ := setupKasBare(t)
	defer mockCtrl.Finish()
	gitlabClient.EXPECT().
		SendUsage(gomock.Any(), gomock.Eq(&gitlab.UsageData{GitopsSyncCount: 5})).
		Return(nil)
	k.usageMetrics.gitopsSyncCount = 5

	// Send accumulated counters
	require.NoError(t, k.sendUsageInternal(ctx))

	// Should not call SendUsage again
	require.NoError(t, k.sendUsageInternal(ctx))
}

func TestSendUsageFailure(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expectedErr := errors.New("expected error")
	k, mockCtrl, _, gitlabClient, errTracker := setupKasBare(t)
	defer mockCtrl.Finish()
	gomock.InOrder(
		gitlabClient.EXPECT().
			SendUsage(gomock.Any(), gomock.Eq(&gitlab.UsageData{GitopsSyncCount: 5})).
			Return(expectedErr),
		errTracker.EXPECT().
			Capture(matcher.ErrorEq("Failed to send usage data: expected error"), gomock.Any()).
			DoAndReturn(func(err error, opts ...errortracking.CaptureOption) {
				cancel() // exception captured, cancel the context to stop the test
			}),
	)
	k.usageMetrics.gitopsSyncCount = 5
	k.usageReportingPeriod = 10 * time.Millisecond

	k.sendUsage(ctx)
}

func TestSendUsageRetry(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k, mockCtrl, _, gitlabClient, _ := setupKasBare(t)
	defer mockCtrl.Finish()
	gomock.InOrder(
		gitlabClient.EXPECT().
			SendUsage(gomock.Any(), gomock.Eq(&gitlab.UsageData{GitopsSyncCount: 5})).
			Return(errors.New("expected error")),
		gitlabClient.EXPECT().
			SendUsage(gomock.Any(), gomock.Eq(&gitlab.UsageData{GitopsSyncCount: 6})).
			Return(nil),
	)
	k.usageMetrics.gitopsSyncCount = 5

	// Try to send accumulated counters, fail
	require.EqualError(t, k.sendUsageInternal(ctx), "expected error")

	k.usageMetrics.gitopsSyncCount++

	// Try again and succeed
	require.NoError(t, k.sendUsageInternal(ctx))

	// Should not call SendUsage again
	require.NoError(t, k.sendUsageInternal(ctx))
}
