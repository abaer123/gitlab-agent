package kas

import (
	"context"
	"errors"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
)

func TestSendUsage(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k, _, _, gitlabClient, _ := setupKasBare(t)
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
	k, _, _, gitlabClient, errTracker := setupKasBare(t)
	errTracker.EXPECT().
		Capture(expectedErr).
		DoAndReturn(func(err error) *sentry.EventID {
			cancel() // exception captured, cancel the context to stop the test
			return nil
		})
	gitlabClient.EXPECT().
		SendUsage(gomock.Any(), gomock.Eq(&gitlab.UsageData{GitopsSyncCount: 5})).
		Return(expectedErr)
	k.usageMetrics.gitopsSyncCount = 5

	k.sendUsage(ctx)
}

func TestSendUsageRetry(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k, _, _, gitlabClient, _ := setupKasBare(t)
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
