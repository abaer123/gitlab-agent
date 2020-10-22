package metric

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/process"
)

const (
	defaultMaxRequestDuration = 15 * time.Second
	shutdownTimeout           = defaultMaxRequestDuration
	readTimeout               = 1 * time.Second
	writeTimeout              = defaultMaxRequestDuration
	idleTimeout               = 1 * time.Minute
)

type Server struct {
	// Name is the name of the application.
	Name          string
	Listener      net.Listener
	UrlPath       string
	Gatherer      prometheus.Gatherer
	Registerer    prometheus.Registerer
	PprofDisabled bool
}

func (a *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Handler:      a.constructHandler(),
		WriteTimeout: writeTimeout,
		ReadTimeout:  readTimeout,
		IdleTimeout:  idleTimeout,
	}
	return process.RunServer(ctx, srv, a.Listener, shutdownTimeout)
}

func (a *Server) constructHandler() http.Handler {
	mux := http.NewServeMux()
	a.maybePprofHandler(mux)
	a.maybePrometheusHandler(mux)
	return mux
}

func (a *Server) setHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", a.Name)
		next.ServeHTTP(w, r)
	})
}

func (a *Server) maybePrometheusHandler(mux *http.ServeMux) {
	if a.Gatherer == nil {
		return
	}
	mux.Handle(
		a.UrlPath,
		a.setHeaders(promhttp.InstrumentMetricHandler(a.Registerer, promhttp.HandlerFor(a.Gatherer, promhttp.HandlerOpts{
			Timeout: defaultMaxRequestDuration,
		}))),
	)
}

func (a *Server) maybePprofHandler(mux *http.ServeMux) {
	if a.PprofDisabled {
		return
	}
	routes := map[string]func(http.ResponseWriter, *http.Request){
		"/debug/pprof/":        pprof.Index,
		"/debug/pprof/cmdline": pprof.Cmdline,
		"/debug/pprof/profile": pprof.Profile,
		"/debug/pprof/symbol":  pprof.Symbol,
		"/debug/pprof/trace":   pprof.Trace,
	}
	for route, handler := range routes {
		mux.Handle(route, a.setHeaders(http.HandlerFunc(handler)))
	}
}
