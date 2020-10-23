package grpctools

import (
	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
)

type ContextAugmenter func(context.Context) (context.Context, error)

// UnaryServerLimitingInterceptor returns a new unary server interceptor that augments connection context.
func UnaryServerCtxAugmentingInterceptor(f ContextAugmenter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx, err := f(ctx)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamServerLimitingInterceptor returns a new stream server interceptor that augments connection context.
func StreamServerCtxAugmentingInterceptor(f ContextAugmenter) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, err := f(ss.Context())
		if err != nil {
			return err
		}
		wrapper := grpc_middleware.WrapServerStream(ss)
		wrapper.WrappedContext = ctx
		return handler(srv, wrapper)
	}
}
