package grpctool

import (
	"context"

	"google.golang.org/grpc"
)

// UnaryClientCtxAugmentingInterceptor returns a new unary client interceptor that augments connection context.
func UnaryClientCtxAugmentingInterceptor(f ContextAugmenter) grpc.UnaryClientInterceptor {
	return func(parentCtx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, err := f(parentCtx)
		if err != nil {
			return err
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientLimitingInterceptor returns a new stream server interceptor that augments connection context.
func StreamClientCtxAugmentingInterceptor(f ContextAugmenter) grpc.StreamClientInterceptor {
	return func(parentCtx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx, err := f(parentCtx)
		if err != nil {
			return nil, err
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}
