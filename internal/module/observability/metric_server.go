package observability

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/httpz"
)

const (
	defaultMaxRequestDuration = 15 * time.Second
	shutdownTimeout           = defaultMaxRequestDuration
	readTimeout               = 1 * time.Second
	writeTimeout              = defaultMaxRequestDuration
	idleTimeout               = 1 * time.Minute
)

type MetricServer struct {
	// Name is the name of the application.
	Name                  string
	Listener              net.Listener
	PrometheusUrlPath     string
	LivenessProbeUrlPath  string
	ReadinessProbeUrlPath string
	Gatherer              prometheus.Gatherer
	Registerer            prometheus.Registerer
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

func (s *MetricServer) setHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", s.Name)
		next.ServeHTTP(w, r)
	})
}

func (s *MetricServer) probesHandler(mux *http.ServeMux) {
	mux.Handle(
		s.LivenessProbeUrlPath,
		s.setHeaders(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)
	mux.Handle(
		s.ReadinessProbeUrlPath,
		s.setHeaders(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)
}

func (s *MetricServer) prometheusHandler(mux *http.ServeMux) {
	mux.Handle(
		s.PrometheusUrlPath,
		s.setHeaders(promhttp.InstrumentMetricHandler(s.Registerer, promhttp.HandlerFor(s.Gatherer, promhttp.HandlerOpts{
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
		mux.Handle(route, s.setHeaders(http.HandlerFunc(handler)))
	}
}
