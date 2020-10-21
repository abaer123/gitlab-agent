package rpclimiter

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ClientLimiter defines the interface to perform client-side request rate limiting.
// You can use golang.org/x/time/rate.Limiter as an implementation of this interface.
type ClientLimiter interface {
	// Wait blocks until limiter permits an event to happen.
	// It returns an error if the Context is
	// canceled, or the expected wait time exceeds the Context's Deadline.
	Wait(context.Context) error
}

// UnaryClientInterceptor returns a new unary client interceptors that performs request rate limiting.
func UnaryClientInterceptor(limiter ClientLimiter) grpc.UnaryClientInterceptor {
	return func(parentCtx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if err := limiter.Wait(parentCtx); err != nil {
			return status.Errorf(codes.ResourceExhausted, "%s is rejected by rpclimiter middleware, please retry later", method)
		}
		return invoker(parentCtx, method, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor returns a new stream server interceptor that performs rate limiting on the request.
func StreamClientInterceptor(limiter ClientLimiter) grpc.StreamClientInterceptor {
	return func(parentCtx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if err := limiter.Wait(parentCtx); err != nil {
			return nil, status.Errorf(codes.ResourceExhausted, "%s is rejected by rpclimiter middleware, please retry later", method)
		}
		return streamer(parentCtx, desc, cc, method, opts...)
	}
}
