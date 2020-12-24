package grpctool

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"go.uber.org/zap"
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

// LoggerInjector can be used to augment a context from an incoming gRPC request with a logger.
// The logger gets a correlation id field from the context.
func LoggerInjector(log *zap.Logger) ContextAugmenter {
	return func(ctx context.Context) (context.Context, error) {
		return InjectLogger(ctx, log.With(logz.CorrelationIdFromContext(ctx))), nil
	}
}
