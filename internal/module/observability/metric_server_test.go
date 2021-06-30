package observability

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_errtracker"
	"go.uber.org/zap/zaptest"
)

func TestMetricServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()
	logger := zaptest.NewLogger(t)
	innerLivenessProbe := NoopProbe
	innerReadinessProbe := NoopProbe
	livenessProbe := func(ctx context.Context) error {
		return innerLivenessProbe(ctx)
	}
	readinessProbe := func(ctx context.Context) error {
		return innerReadinessProbe(ctx)
	}
	tracker := mock_errtracker.NewMockTracker(ctrl)
	metricServer := &MetricServer{
		Tracker:               tracker,
		Log:                   logger,
		Name:                  "test-server",
		Listener:              listener,
		PrometheusUrlPath:     "/metrics",
		LivenessProbeUrlPath:  "/liveness",
		ReadinessProbeUrlPath: "/readiness",
		Gatherer:              prometheus.DefaultGatherer,
		Registerer:            prometheus.DefaultRegisterer,
		LivenessProbe:         livenessProbe,
		ReadinessProbe:        readinessProbe,
	}
	handler := metricServer.constructHandler()

	httpGet := func(t *testing.T, path string) *httptest.ResponseRecorder {
		request, err := http.NewRequest("GET", path, nil) // nolint:noctx
		require.NoError(t, err)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		return recorder
	}

	// tests

	t.Run("/metrics", func(t *testing.T) {
		httpResponse := httpGet(t, "/metrics").Result()
		require.Equal(t, http.StatusOK, httpResponse.StatusCode)
		httpResponse.Body.Close()
	})

	t.Run("/liveness", func(t *testing.T) {
		rec := httpGet(t, "/liveness")
		httpResponse := rec.Result()
		require.Equal(t, http.StatusOK, httpResponse.StatusCode)
		require.Empty(t, rec.Body)
		httpResponse.Body.Close()

		expectedErr := fmt.Errorf("failed liveness on purpose")
		innerLivenessProbe = func(context.Context) error {
			return expectedErr
		}
		tracker.EXPECT().Capture(fmt.Errorf("LivenessProbe failed: %w", expectedErr), gomock.Any())

		rec = httpGet(t, "/liveness")
		httpResponse = rec.Result()
		require.Equal(t, http.StatusInternalServerError, httpResponse.StatusCode)
		require.Equal(t, "failed liveness on purpose", rec.Body.String())
		httpResponse.Body.Close()
	})

	t.Run("/readiness", func(t *testing.T) {
		rec := httpGet(t, "/readiness")
		httpResponse := rec.Result()
		require.Equal(t, http.StatusOK, httpResponse.StatusCode)
		require.Empty(t, rec.Body)
		httpResponse.Body.Close()

		expectedErr := fmt.Errorf("failed readiness on purpose")
		innerReadinessProbe = func(context.Context) error {
			return expectedErr
		}
		tracker.EXPECT().Capture(fmt.Errorf("ReadinessProbe failed: %w", expectedErr), gomock.Any())

		rec = httpGet(t, "/readiness")
		httpResponse = rec.Result()
		require.Equal(t, http.StatusInternalServerError, httpResponse.StatusCode)
		require.Equal(t, "failed readiness on purpose", rec.Body.String())
		httpResponse.Body.Close()
	})
}
