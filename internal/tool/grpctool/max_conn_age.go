package grpctool

import (
	"context"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/mathz"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/stats"
)

// Load balancer in front of kas typically has a timeout parameter, limiting the maximum connection age.
// After it elapses, the load balancer aborts the connection and the client sees a connection reset.
// This is not good. To mitigate the problem all long running server-side poll loops should stop beforehand.
//
// After gRPC max connection age time elapses, gRPC will send a HTTP/2 GOAWAY frame to the client, to let it know that it
// should not use this TCP connection for any new RPCs and, after all in-flight requests had finished, close it.
// After gRPC max connection age time elapses, max connection grace period starts, allowing in-flight requests to finish cleanly.
// After this second time interval elapses, gRPC aborts the connection.
// https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/138 is the example of the above
// behavior.
// We tag each TCP connection with a timeout context set to expire after gRPC max connection age duration to be able to
// return from the request handler just after gRPC max connection age duration had elapsed. Returning after and not
// before is important so that the client receives GOAWAY first and makes no further requests on this TCP connection.

// https://github.com/grpc/grpc/issues/26703 feature request to help get rid of the messy stuff below.

type maxConnAgeCtxKeyType int

const maxConnAgeCtxKey maxConnAgeCtxKeyType = iota

const (
	maxConnectionAgeJitterPercent = 5
	// gRPC applies +/- 10% jitter to MaxConnectionAge parameter.
	// https://github.com/grpc/grpc-go/blob/v1.39.0/internal/transport/http2_server.go#L1339-L1347
	maxConnectionAgeGrpcJitterPercent = 10
	maxConnectionAgeGracePercent      = 20
)

func MaxConnectionAge2GrpcKeepalive(auxCtx context.Context, maxConnectionAge time.Duration) (grpc.ServerOption, stats.Handler) {
	kp, sh := maxConnectionAge2GrpcKeepalive(auxCtx, maxConnectionAge)
	return grpc.KeepaliveParams(kp), sh
}

func maxConnectionAge2GrpcKeepalive(auxCtx context.Context, maxConnectionAge time.Duration) (keepalive.ServerParameters, stats.Handler) {
	// See https://github.com/grpc/grpc-go/blob/v1.33.1/internal/transport/http2_server.go#L949-L1047
	// to better understand how keepalive works.
	kp := keepalive.ServerParameters{
		MaxConnectionAge: maxConnectionAge * (100 - maxConnectionAgeGracePercent - maxConnectionAgeGrpcJitterPercent - maxConnectionAgeJitterPercent) / 100, // nolint: durationcheck
		// Give pending RPCs some time to complete.
		MaxConnectionAgeGrace: maxConnectionAge * maxConnectionAgeGracePercent / 100,
		// Trying to stay below 60 seconds (typical load-balancer timeout)
		Time: 50 * time.Second,
	}
	sh := NewMaxConnAgeStatsHandler(auxCtx, maxConnectionAge*(100-maxConnectionAgeGracePercent-maxConnectionAgeJitterPercent)/100) // nolint: durationcheck
	return kp, sh
}

func MaxConnectionAgeContextFromStream(stream grpc.ServerStream) context.Context {
	return stream.Context().Value(maxConnAgeCtxKey).(context.Context)
}

func AddMaxConnectionAgeContext(ctx, ageCtx context.Context) context.Context {
	return context.WithValue(ctx, maxConnAgeCtxKey, ageCtx)
}

type joinStatHandlers struct {
	handlers []stats.Handler
}

func NewJoinStatHandlers(handlers ...stats.Handler) stats.Handler {
	return joinStatHandlers{
		handlers: handlers,
	}
}

func (h joinStatHandlers) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	for _, c := range h.handlers {
		ctx = c.TagRPC(ctx, info)
	}
	return ctx
}

func (h joinStatHandlers) HandleRPC(ctx context.Context, rpcStats stats.RPCStats) {
	for _, c := range h.handlers {
		c.HandleRPC(ctx, rpcStats)
	}
}

func (h joinStatHandlers) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	for _, c := range h.handlers {
		ctx = c.TagConn(ctx, info)
	}
	return ctx
}

func (h joinStatHandlers) HandleConn(ctx context.Context, connStats stats.ConnStats) {
	for _, c := range h.handlers {
		c.HandleConn(ctx, connStats)
	}
}

type maxConnAgeStatsHandler struct {
	auxCtx           context.Context
	maxConnectionAge time.Duration
}

func NewMaxConnAgeStatsHandler(auxCtx context.Context, maxConnectionAge time.Duration) stats.Handler {
	return maxConnAgeStatsHandler{
		auxCtx:           auxCtx,
		maxConnectionAge: maxConnectionAge,
	}
}

func (m maxConnAgeStatsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return ctx
}

func (m maxConnAgeStatsHandler) HandleRPC(ctx context.Context, rpcStats stats.RPCStats) {
}

func (m maxConnAgeStatsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	var (
		ageCtx    context.Context
		ageCancel context.CancelFunc
	)
	if m.maxConnectionAge == 0 {
		ageCtx, ageCancel = context.WithCancel(m.auxCtx)
	} else {
		ageCtx, ageCancel = context.WithTimeout(m.auxCtx, mathz.DurationWithPositiveJitter(m.maxConnectionAge, maxConnectionAgeJitterPercent))
	}
	go func() {
		select {
		case <-ageCtx.Done():
		case <-ctx.Done():
			ageCancel()
		}
	}()
	return AddMaxConnectionAgeContext(ctx, ageCtx)
}

func (m maxConnAgeStatsHandler) HandleConn(ctx context.Context, connStats stats.ConnStats) {
}
