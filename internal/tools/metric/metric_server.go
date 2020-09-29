package metric

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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
	Name     string
	Listener net.Listener
	UrlPath  string
	Gatherer prometheus.Gatherer
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

func (a *Server) constructHandler() *mux.Router {
	router := mux.NewRouter()
	router.Use(a.setServerHeader)
	router.Methods(http.MethodGet).
		Path(a.UrlPath).
		Handler(promhttp.HandlerFor(a.Gatherer, promhttp.HandlerOpts{
			Timeout: defaultMaxRequestDuration,
		}))
	return router
}

func (a *Server) setServerHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", a.Name)
		next.ServeHTTP(w, r)
	})
}
