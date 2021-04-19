package grpctool

import (
	"context"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type loggerKeyType int

const (
	loggerKey loggerKeyType = iota
)

func InjectLogger(ctx context.Context, log *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, log)
}

func LoggerFromContext(ctx context.Context) *zap.Logger {
	log, ok := ctx.Value(loggerKey).(*zap.Logger)
	if !ok {
		// This is a programmer error, so panic.
		panic("*zap.Logger not attached to context. Make sure you are using interceptors")
	}
	return log
}

// UnaryServerLoggerInterceptor returns a new unary server interceptor that augments connection context with a logger.
// The logger gets a correlation id field from the context, gRPC service and gRPC method.
func UnaryServerLoggerInterceptor(log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(augmentContextWithLogger(ctx, info.FullMethod, log), req)
	}
}

// StreamServerLoggerInterceptor returns a new stream server interceptor that augments connection context with a logger.
// The logger gets a correlation id field from the context, gRPC service and gRPC method.
func StreamServerLoggerInterceptor(log *zap.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapper := grpc_middleware.WrapServerStream(ss)
		wrapper.WrappedContext = augmentContextWithLogger(wrapper.WrappedContext, info.FullMethod, log)
		return handler(srv, wrapper)
	}
}

func augmentContextWithLogger(ctx context.Context, fullMethodName string, log *zap.Logger) context.Context {
	service, method := splitMethod(fullMethodName)
	return InjectLogger(ctx, log.With(
		logz.CorrelationIdFromContext(ctx),
		logz.GrpcService(service),
		logz.GrpcMethod(method),
	))
}

func splitMethod(fullMethodName string) (string /* service */, string /* method */) {
	if fullMethodName != "" && fullMethodName[0] == '/' {
		fullMethodName = fullMethodName[1:]
	}
	pos := strings.LastIndex(fullMethodName, "/")
	if pos == -1 {
		return "unknown", fullMethodName
	}
	service := fullMethodName[:pos]
	method := fullMethodName[pos+1:]
	return service, method
}
