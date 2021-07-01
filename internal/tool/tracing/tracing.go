package tracing

import (
	"fmt"
	"io"

	"github.com/opentracing/opentracing-go"
	"gitlab.com/gitlab-org/labkit/tracing/connstr"
	"gitlab.com/gitlab-org/labkit/tracing/impl"
)

// ConstructTracer relies on LabKit for tracing implementations.
func ConstructTracer(serviceName, connectionString string) (opentracing.Tracer, io.Closer, error) {
	if connectionString == "" {
		// No opentracing connection has been set
		return opentracing.NoopTracer{}, nopCloser{}, nil
	}

	driverName, options, err := connstr.Parse(connectionString)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse connection: %w", err)
	}

	if serviceName != "" {
		options["ServiceName"] = serviceName
	}

	tracer, closer, err := impl.New(driverName, options)
	if err != nil {
		return nil, nil, err
	}
	if closer == nil {
		closer = nopCloser{}
	}
	return tracer, closer, nil
}

type nopCloser struct {
}

func (nopCloser) Close() error { return nil }
