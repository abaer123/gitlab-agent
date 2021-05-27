package server

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/testhelpers"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/kascfg"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	_ modserver.Module        = &module{}
	_ modserver.Factory       = &Factory{}
	_ modserver.ApplyDefaults = ApplyDefaults
)

func TestSendUsage(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	counters := map[string]int64{
		"x": 5,
	}
	m, tracker, _ := setupModule(t, func(w http.ResponseWriter, r *http.Request) {
		assertNoContentRequest(t, r, counters)
		w.WriteHeader(http.StatusNoContent)
	})
	ud := &usage_metrics.UsageData{Counters: counters}
	gomock.InOrder(
		tracker.EXPECT().
			CloneUsageData().
			Return(ud),
		tracker.EXPECT().
			Subtract(ud),
		tracker.EXPECT().
			CloneUsageData().
			DoAndReturn(func() *usage_metrics.UsageData {
				cancel()
				return &usage_metrics.UsageData{}
			}),
	)
	require.NoError(t, m.Run(ctx))
}

func TestSendUsageFailureAndRetry(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	counters1 := map[string]int64{
		"x": 5,
	}
	ud1 := &usage_metrics.UsageData{Counters: counters1}
	counters2 := map[string]int64{
		"x": 6,
	}
	ud2 := &usage_metrics.UsageData{Counters: counters2}
	var call int
	m, tracker, mockApi := setupModule(t, func(w http.ResponseWriter, r *http.Request) {
		call++
		switch call {
		case 1:
			assertNoContentRequest(t, r, counters1)
			w.WriteHeader(http.StatusInternalServerError)
		case 2:
			assertNoContentRequest(t, r, counters2)
			w.WriteHeader(http.StatusNoContent)
		default:
			assert.Fail(t, "unexpected call", call)
		}
	})
	gomock.InOrder(
		tracker.EXPECT().
			CloneUsageData().
			Return(ud1),
		mockApi.EXPECT().
			HandleProcessingError(gomock.Any(), gomock.Any(), "Failed to send usage data", matcher.ErrorEq("error kind: 0; status: 500")),
		tracker.EXPECT().
			CloneUsageData().
			Return(ud2),
		tracker.EXPECT().
			Subtract(ud2),
		tracker.EXPECT().
			CloneUsageData().
			DoAndReturn(func() *usage_metrics.UsageData {
				cancel()
				return &usage_metrics.UsageData{}
			}),
	)
	require.NoError(t, m.Run(ctx))
}

func TestSendUsageHttp(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	counters := map[string]int64{
		"x": 5,
	}
	ud := &usage_metrics.UsageData{Counters: counters}

	m, tracker, _ := setupModule(t, func(w http.ResponseWriter, r *http.Request) {
		assertNoContentRequest(t, r, counters)
		w.WriteHeader(http.StatusNoContent)
	})

	gomock.InOrder(
		tracker.EXPECT().
			CloneUsageData().
			Return(ud),
		tracker.EXPECT().
			Subtract(ud).
			Do(func(ud *usage_metrics.UsageData) {
				cancel()
			}),
	)
	require.NoError(t, m.Run(ctx))
}

func setupModule(t *testing.T, handler func(http.ResponseWriter, *http.Request)) (*module, *mock_usage_metrics.MockUsageTrackerInterface, *mock_modserver.MockAPI) {
	ctrl := gomock.NewController(t)
	tracker := mock_usage_metrics.NewMockUsageTrackerInterface(ctrl)
	mockApi := mock_modserver.NewMockAPI(ctrl)
	f := Factory{
		UsageTracker: tracker,
	}
	config := &kascfg.ConfigurationFile{}
	ApplyDefaults(config)
	config.Observability.UsageReportingPeriod = durationpb.New(50 * time.Millisecond)
	m, err := f.New(&modserver.Config{
		Log:          zaptest.NewLogger(t),
		Api:          mockApi,
		Config:       config,
		GitLabClient: mock_gitlab.SetupClient(t, usagePingApiPath, handler),
		UsageTracker: tracker,
	})
	require.NoError(t, err)
	return m.(*module), tracker, mockApi
}

func assertNoContentRequest(t *testing.T, r *http.Request, expectedPayload interface{}) {
	testhelpers.AssertRequestMethod(t, r, http.MethodPost)
	assert.Empty(t, r.Header.Values("Accept"))
	testhelpers.AssertRequestContentTypeJson(t, r)
	testhelpers.AssertRequestUserAgent(t, r, testhelpers.KasUserAgent)
	assert.Equal(t, testhelpers.KasCorrelationClientName, r.Header.Get(testhelpers.CorrelationClientNameHeader))
	testhelpers.AssertJWTSignature(t, r)
	expectedBin, err := json.Marshal(expectedPayload)
	if !assert.NoError(t, err) {
		return
	}
	var expected interface{}
	err = json.Unmarshal(expectedBin, &expected)
	if !assert.NoError(t, err) {
		return
	}
	actualBin, err := ioutil.ReadAll(r.Body)
	if !assert.NoError(t, err) {
		return
	}
	var actual interface{}
	err = json.Unmarshal(actualBin, &actual)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, expected, actual)
}
