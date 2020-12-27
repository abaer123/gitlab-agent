package server

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
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
			DoJSON(ctx, http.MethodPost, usagePingApiPath, nil, api.AgentToken(""), counters, nil),
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
			DoJSON(ctx, http.MethodPost, usagePingApiPath, nil, api.AgentToken(""), counters1, nil).
			Return(expectedErr),
		mockApi.EXPECT().
			HandleProcessingError(gomock.Any(), gomock.Any(), "Failed to send usage data", expectedErr),
		tracker.EXPECT().
			CloneUsageData().
			Return(ud2, false),
		client.EXPECT().
			DoJSON(ctx, http.MethodPost, usagePingApiPath, nil, api.AgentToken(""), counters2, nil),
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

func TestSendUsageHttp(t *testing.T) {
	ctx, correlationId := mock_gitlab.CtxWithCorrelation(t)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	counters := map[string]int64{
		"x": 5,
	}
	ud := &usage_metrics.UsageData{Counters: counters}
	r := http.NewServeMux()
	r.HandleFunc(usagePingApiPath, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		mock_gitlab.AssertCommonRequestParams(t, r, correlationId)
		if !mock_gitlab.AssertJWTSignature(t, w, r) {
			return
		}
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		data, err := ioutil.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var req map[string]int64
		err = json.Unmarshal(data, &req)
		if !assert.NoError(t, err) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		assert.Empty(t, cmp.Diff(req, counters))

		w.WriteHeader(http.StatusNoContent)
	})
	s := httptest.NewServer(r)
	defer s.Close()

	u, err := url.Parse(s.URL)
	require.NoError(t, err)

	m, tracker, _, _ := setupModule(t)
	m.gitLabClient = gitlab.NewClient(u, []byte(mock_gitlab.AuthSecretKey), mock_gitlab.ClientOptionsForTest()...)

	gomock.InOrder(
		tracker.EXPECT().
			CloneUsageData().
			Return(ud, false),
		tracker.EXPECT().
			Subtract(ud).Do(func(ud *usage_metrics.UsageData) {
			cancel()
		}),
	)
	require.NoError(t, m.Run(ctx))
}

func setupModule(t *testing.T) (*module, *mock_usage_metrics.MockUsageTrackerInterface, *mock_gitlab.MockClientInterface, *mock_modserver.MockAPI) {
	ctrl := gomock.NewController(t)
	tracker := mock_usage_metrics.NewMockUsageTrackerInterface(ctrl)
	client := mock_gitlab.NewMockClientInterface(ctrl)
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
		GitLabClient: client,
		UsageTracker: tracker,
	})
	require.NoError(t, err)
	return m.(*module), tracker, client, mockApi
}
