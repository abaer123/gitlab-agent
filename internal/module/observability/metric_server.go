package observability

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/httpz"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"go.uber.org/zap"
)

const (
	defaultMaxRequestDuration = 15 * time.Second
	shutdownTimeout           = defaultMaxRequestDuration
	readTimeout               = 1 * time.Second
	writeTimeout              = defaultMaxRequestDuration
	idleTimeout               = 1 * time.Minute
)

// Probe is the expected type for probe functions
type Probe func(context.Context) error

// NoopProbe is a placeholder probe for convenience
func NoopProbe(context.Context) error {
	return nil
}

func ChainProbes(probes ...Probe) Probe {
	return func(ctx context.Context) error {
		for _, probe := range probes {
			err := probe(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

type MetricServer struct {
	Tracker errortracking.Tracker
	Log     *zap.Logger
	// Name is the name of the application.
	Name                  string
	Listener              net.Listener
	PrometheusUrlPath     string
	LivenessProbeUrlPath  string
	ReadinessProbeUrlPath string
	Gatherer              prometheus.Gatherer
	Registerer            prometheus.Registerer
	LivenessProbe         Probe
	ReadinessProbe        Probe
}

func (s *MetricServer) Run(ctx context.Context) error {
	srv := &http.Server{
		Handler:      s.constructHandler(),
		WriteTimeout: writeTimeout,
		ReadTimeout:  readTimeout,
		IdleTimeout:  idleTimeout,
	}
	return httpz.RunServer(ctx, srv, s.Listener, shutdownTimeout)
}

func (s *MetricServer) constructHandler() http.Handler {
	mux := http.NewServeMux()
	s.probesHandler(mux)
	s.pprofHandler(mux)
	s.prometheusHandler(mux)
	return mux
}

func (s *MetricServer) setHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", s.Name)
		next.ServeHTTP(w, r)
	})
}

func (s *MetricServer) probesHandler(mux *http.ServeMux) {
	mux.Handle(
		s.LivenessProbeUrlPath,
		s.setHeader(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			err := s.LivenessProbe(request.Context())
			if err != nil {
				s.logAndCapture(request.Context(), "LivenessProbe failed", err)
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = io.WriteString(w, err.Error())
				return
			}
			w.WriteHeader(http.StatusOK)
		})),
	)
	mux.Handle(
		s.ReadinessProbeUrlPath,
		s.setHeader(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			err := s.ReadinessProbe(request.Context())
			if err != nil {
				s.logAndCapture(request.Context(), "ReadinessProbe failed", err)
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = io.WriteString(w, err.Error())
				return
			}
			w.WriteHeader(http.StatusOK)
		})),
	)
}

func (s *MetricServer) prometheusHandler(mux *http.ServeMux) {
	mux.Handle(
		s.PrometheusUrlPath,
		s.setHeader(promhttp.InstrumentMetricHandler(s.Registerer, promhttp.HandlerFor(s.Gatherer, promhttp.HandlerOpts{
			Timeout: defaultMaxRequestDuration,
		}))),
	)
}

func (s *MetricServer) pprofHandler(mux *http.ServeMux) {
	routes := map[string]func(http.ResponseWriter, *http.Request){
		"/debug/pprof/":        pprof.Index,
		"/debug/pprof/cmdline": pprof.Cmdline,
		"/debug/pprof/profile": pprof.Profile,
		"/debug/pprof/symbol":  pprof.Symbol,
		"/debug/pprof/trace":   pprof.Trace,
	}
	for route, handler := range routes {
		mux.Handle(route, s.setHeader(http.HandlerFunc(handler)))
	}
}

func (s *MetricServer) logAndCapture(ctx context.Context, msg string, err error) {
	s.Log.Error(msg, zap.Error(err))
	s.Tracker.Capture(fmt.Errorf("%s: %v", msg, err), errortracking.WithContext(ctx))
}
