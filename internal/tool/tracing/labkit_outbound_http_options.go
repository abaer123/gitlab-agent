package tracing

import (
	"fmt"
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
)

// OperationNamer will return an operation name given an HTTP request
type OperationNamer func(*http.Request) string

// The configuration for InjectCorrelationID
type roundTripperConfig struct {
	getOperationName OperationNamer
	tracer           opentracing.Tracer
	log              *zap.Logger
}

// RoundTripperOption will configure a correlation handler
type RoundTripperOption func(*roundTripperConfig)

func applyRoundTripperOptions(opts []RoundTripperOption) roundTripperConfig {
	config := roundTripperConfig{
		getOperationName: func(req *http.Request) string {
			// By default use `GET https://localhost` for operation names
			return fmt.Sprintf("%s %s://%s", req.Method, req.URL.Scheme, req.URL.Host)
		},
		tracer: opentracing.GlobalTracer(),
		log:    zap.NewNop(),
	}
	for _, v := range opts {
		v(&config)
	}

	return config
}

// WithRoundTripperTracer sets a custom tracer to be used for this middleware, otherwise the opentracing.GlobalTracer is used.
func WithRoundTripperTracer(tracer opentracing.Tracer) RoundTripperOption {
	return func(config *roundTripperConfig) {
		config.tracer = tracer
	}
}

// WithLogger sets the log to use.
func WithLogger(log *zap.Logger) RoundTripperOption {
	return func(config *roundTripperConfig) {
		config.log = log
	}
}
