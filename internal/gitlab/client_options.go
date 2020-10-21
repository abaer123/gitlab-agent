package gitlab

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
)

// clientConfig holds configuration for the client.
type clientConfig struct {
	tracer      opentracing.Tracer
	log         *zap.Logger
	dialContext func(ctx context.Context, network, address string) (net.Conn, error)
	proxy       func(*http.Request) (*url.URL, error)
	clientName  string
	userAgent   string
}

// ClientOption to configure the client.
type ClientOption func(*clientConfig)

func applyClientOptions(opts []ClientOption) clientConfig {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	config := clientConfig{
		tracer:      opentracing.GlobalTracer(),
		log:         zap.NewNop(),
		dialContext: dialer.DialContext,
		proxy:       http.ProxyFromEnvironment,
		clientName:  "",
		userAgent:   "",
	}
	for _, v := range opts {
		v(&config)
	}

	return config
}

// WithTracer sets a custom tracer to be used, otherwise the opentracing.GlobalTracer is used.
func WithTracer(tracer opentracing.Tracer) ClientOption {
	return func(config *clientConfig) {
		config.tracer = tracer
	}
}

// WithCorrelationClientName configures the X-GitLab-Client-Name header on the http client.
func WithCorrelationClientName(clientName string) ClientOption {
	return func(config *clientConfig) {
		config.clientName = clientName
	}
}

// WithUserAgent configures the User-Agent header on the http client.
func WithUserAgent(userAgent string) ClientOption {
	return func(config *clientConfig) {
		config.userAgent = userAgent
	}
}

// WithLogger sets the log to use.
func WithLogger(log *zap.Logger) ClientOption {
	return func(config *clientConfig) {
		config.log = log
	}
}
