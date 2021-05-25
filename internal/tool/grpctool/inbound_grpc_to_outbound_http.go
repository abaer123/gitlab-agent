package grpctool

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/prototool"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	headerFieldNumber  protoreflect.FieldNumber = 1
	dataFieldNumber    protoreflect.FieldNumber = 2
	trailerFieldNumber protoreflect.FieldNumber = 3

	maxDataChunkSize = 32 * 1024
)

type InboundGrpcToOutboundHttpStream interface {
	Send(*HttpResponse) error
	grpc.ServerStream
}

// API is a reduced version on modshared.API.
// It's here to avoid the dependency.
type API interface {
	HandleProcessingError(ctx context.Context, log *zap.Logger, msg string, err error)
	HandleSendError(log *zap.Logger, msg string, err error) error
}

type HttpDo func(ctx context.Context, header *HttpRequest_Header, body io.Reader) (*http.Response, error)

type InboundGrpcToOutboundHttp struct {
	streamVisitor *StreamVisitor
	api           API
	httpDo        HttpDo
}

func NewInboundGrpcToOutboundHttp(api API, httpDo HttpDo) *InboundGrpcToOutboundHttp {
	sv, err := NewStreamVisitor(&HttpRequest{})
	if err != nil {
		panic(err) // this will never panic as long as the proto file is correct
	}
	return &InboundGrpcToOutboundHttp{
		streamVisitor: sv,
		api:           api,
		httpDo:        httpDo,
	}
}

func (x *InboundGrpcToOutboundHttp) Pipe(server InboundGrpcToOutboundHttpStream) error {
	ctx := server.Context()
	log := LoggerFromContext(ctx)

	g, ctx := errgroup.WithContext(ctx) // if one of the goroutines returns a non-nil error, ctx is canceled.

	pr, pw := io.Pipe()
	headerMsg := make(chan *HttpRequest_Header)

	// Pipe gRPC request -> HTTP request
	g.Go(func() error {
		return x.pipeGrpcIntoHttp(ctx, server, headerMsg, pw)
	})
	// Pipe HTTP response -> gRPC response
	g.Go(func() error {
		// Make sure the writer is unblocked if we exit abruptly
		// The error is ignored because it will always occur if things go normally - the pipe will have been
		// closed already when this code is reached (and that's an error).
		defer pr.Close() // nolint: errcheck
		select {
		case <-ctx.Done():
			return ctx.Err()
		case header := <-headerMsg:
			resp, err := x.httpDo(ctx, header, pr)
			if err != nil {
				return err
			}
			return x.pipeHttpIntoGrpc(log, server, resp)
		}
	})

	err := g.Wait()
	switch {
	case err == nil:
	case errz.ContextDone(err):
		err = status.Error(codes.Unavailable, "unavailable")
	case IsStatusError(err):
		// A gRPC status already
	default:
		x.api.HandleProcessingError(ctx, log, "gRPC -> HTTP", err)
		err = status.Error(codes.Unavailable, "unavailable")
	}
	return err
}

func (x *InboundGrpcToOutboundHttp) pipeGrpcIntoHttp(ctx context.Context, server grpc.ServerStream, headerMsg chan *HttpRequest_Header, pw *io.PipeWriter) error {
	return x.streamVisitor.Visit(server,
		WithCallback(headerFieldNumber, func(header *HttpRequest_Header) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case headerMsg <- header:
				return nil
			}
		}),
		WithCallback(dataFieldNumber, func(data *HttpRequest_Data) error {
			_, err := pw.Write(data.Data)
			return err
		}),
		WithCallback(trailerFieldNumber, func(trailer *HttpRequest_Trailer) error {
			// Nothing to do
			return nil
		}),
		WithEOFCallback(pw.Close), // Nothing more to send, close the write end of the pipe
	)
}

func (x *InboundGrpcToOutboundHttp) pipeHttpIntoGrpc(log *zap.Logger, server grpc.ServerStream, resp *http.Response) error {
	err := func() (retErr error) { // closure to close resp.Body ASAP
		defer errz.SafeClose(resp.Body, &retErr)
		err := server.SendMsg(&HttpResponse{
			Message: &HttpResponse_Header_{
				Header: &HttpResponse_Header{
					Response: &prototool.HttpResponse{
						StatusCode: int32(resp.StatusCode),
						Status:     resp.Status,
						Header:     prototool.HttpHeaderToValuesMap(resp.Header),
					},
				},
			},
		})
		if err != nil {
			return x.api.HandleSendError(log, "Failed to send HTTP header", err)
		}

		buffer := make([]byte, maxDataChunkSize)
		for err == nil { // loop while not EOF
			var n int
			n, err = resp.Body.Read(buffer)
			if err != nil && !errors.Is(err, io.EOF) {
				return fmt.Errorf("read response body: %w", err) // wrap
			}
			if n > 0 { // handle n=0, err=io.EOF case
				sendErr := server.SendMsg(&HttpResponse{
					Message: &HttpResponse_Data_{
						Data: &HttpResponse_Data{
							Data: buffer[:n],
						},
					},
				})
				if sendErr != nil {
					return x.api.HandleSendError(log, "Failed to send HTTP data", sendErr)
				}
			}
		}
		return nil
	}()
	if err != nil {
		return err
	}

	err = server.SendMsg(&HttpResponse{
		Message: &HttpResponse_Trailer_{
			Trailer: &HttpResponse_Trailer{},
		},
	})
	if err != nil {
		return x.api.HandleSendError(log, "Failed to send HTTP trailer", err)
	}
	return nil
}
