package kasapp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/mathz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/retry"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	routeAttemptPeriod      = 3 * time.Second
	getTunnelsAttemptPeriod = 1 * time.Second

	tunnelReadyFieldNumber protoreflect.FieldNumber = 1
	headerFieldNumber      protoreflect.FieldNumber = 2
	messageFieldNumber     protoreflect.FieldNumber = 3
	trailerFieldNumber     protoreflect.FieldNumber = 4
)

var (
	proxyStreamDesc = grpc.StreamDesc{
		ServerStreams: true,
		ClientStreams: true,
	}
)

// RouteToCorrectKasHandler is a gRPC handler that routes the request to another kas instance.
// Must return a gRPC status-compatible error.
func (r *router) RouteToCorrectKasHandler(srv interface{}, stream grpc.ServerStream) error {
	ctx := stream.Context()
	md, _ := metadata.FromIncomingContext(ctx)
	agentId, err := agentIdFromMeta(md)
	if err != nil {
		return err
	}
	log := grpctool.LoggerFromContext(ctx).With(logz.AgentId(agentId))
	return r.routeToCorrectKas(log, agentId, stream)
}

// routeToCorrectKas
// must return a gRPC status-compatible error.
func (r *router) routeToCorrectKas(log *zap.Logger, agentId int64, stream grpc.ServerStream) error {
	err := retry.PollImmediateUntil(stream.Context(), routeAttemptPeriod, r.attemptToRoute(log, agentId, stream))
	if errors.Is(err, retry.ErrWaitTimeout) {
		return status.Error(codes.Unavailable, "Unavailable")
	}
	return err // nil or some gRPC status value
}

// attemptToRoute
// must return a gRPC status-compatible error or retry.ErrWaitTimeout.
func (r *router) attemptToRoute(log *zap.Logger, agentId int64, stream grpc.ServerStream) retry.ConditionFunc {
	ctx := stream.Context()
	return func() (bool /* done */, error) {
		var tunnels []*tracker.TunnelInfo
		err := retry.PollImmediateUntil(ctx, getTunnelsAttemptPeriod, r.attemptToGetTunnels(ctx, log, agentId, &tunnels))
		if err != nil {
			return false, err
		}
		mathz.Shuffle(len(tunnels), func(i, j int) {
			tunnels[i], tunnels[j] = tunnels[j], tunnels[i]
		})
		for _, tunnel := range tunnels {
			// Redefines log variable to eliminate the chance of using the original one
			log := log.With(logz.ConnectionId(tunnel.ConnectionId), logz.KasUrl(tunnel.KasUrl)) // nolint:govet
			log.Debug("Trying tunnel")
			err, done := r.attemptToRouteViaTunnel(log, tunnel, stream)
			switch {
			case done:
				// Request was routed successfully. The remote may have returned an error, but that's still a
				// successful response as far as we are concerned. Our job it to route the request and return what
				// the remote responded with.
				return true, err
			case err == nil:
				// No error to log, but also not a success. Continue to try the next tunnel.
			case errors.Is(err, context.Canceled):
				return false, status.Error(codes.Canceled, err.Error())
			case errors.Is(err, context.DeadlineExceeded):
				return false, status.Error(codes.DeadlineExceeded, err.Error())
			default:
				// There was an error routing the request via this tunnel. Log and try another one.
				log.Error("Failed to route request", zap.Error(err))
			}
		}
		return false, nil
	}
}

// attemptToGetTunnels
// must return a gRPC status-compatible error or retry.ErrWaitTimeout.
func (r *router) attemptToGetTunnels(ctx context.Context, log *zap.Logger, agentId int64, infosTarget *[]*tracker.TunnelInfo) retry.ConditionFunc {
	return func() (bool /* done */, error) {
		var infos tunnelInfoCollector
		err := r.tunnelQuerier.GetTunnelsByAgentId(ctx, agentId, infos.Collect)
		if err != nil {
			// TODO error tracking
			log.Error("GetTunnelsByAgentId()", zap.Error(err))
			return false, nil // don't return an error, keep trying
		}
		*infosTarget = infos
		return true, nil
	}
}

// attemptToRouteViaTunnel attempts to route the stream via the tunnel.
// Unusual signature to signal that the done bool should be checked to determine what the error value means.
func (r *router) attemptToRouteViaTunnel(log *zap.Logger, tunnel *tracker.TunnelInfo, stream grpc.ServerStream) (error, bool /* done */) {
	ctx := stream.Context()
	kasClient, err := r.kasPool.Dial(ctx, tunnel.KasUrl)
	if err != nil {
		return err, false
	}
	defer kasClient.Close() // nolint:errcheck
	md, _ := metadata.FromIncomingContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // ensure outbound stream is cleaned up
	kasStream, err := kasClient.NewStream(
		metadata.NewOutgoingContext(ctx, md),
		&proxyStreamDesc,
		grpc.ServerTransportStreamFromContext(ctx).Method(),
		grpc.ForceCodec(grpctool.RawCodecWithProtoFallback{}),
	)
	if err != nil {
		return fmt.Errorf("NewStream(): %w", err), false
	}

	// The gateway kas will block until it has a matching tunnel if it does not have one already. Yes, we found the
	// "correct" kas by looking at Redis, but that tunnel may be no longer available (e.g. disconnected or
	// has been used by another request) and there might not be any other matching tunnels.
	// So in the future we may establish connections to one or more other suitable kas instances concurrently (after
	// a small delay or immediately) to ensure the stream is routed to an agent ASAP.
	// To ensure that the stream is only routed to a single agent, we need the gateway kas to tell us (the routing kas)
	// that it has a matching tunnel. That way we know to which connection to forward the stream to.
	// We then need to tell the gateway kas that we are starting to route the stream to it. If we don't and just
	// close the connection, it does not have to use the tunnel it found and can put it back into it's
	// "ready to be used" list of tunnels.

	var kasResponse GatewayKasResponse
	err = kasStream.RecvMsg(&kasResponse) // Wait for the tunnel to be found
	if err != nil {
		if errors.Is(err, io.EOF) {
			// Gateway kas closed the connection cleanly, perhaps it's been open for too long
			return nil, false
		}
		return fmt.Errorf("kas RecvMsg(): %w", err), false
	}
	tunnelReady := kasResponse.GetTunnelReady()
	if tunnelReady == nil {
		return fmt.Errorf("invalid oneof value type: %T", kasResponse.Msg), false
	}
	err = kasStream.SendMsg(&StartStreaming{})
	if err != nil {
		return fmt.Errorf("stream SendMsg(): %w", err), false
	}
	return r.forwardStream(log, kasStream, stream), true
}

// forwardStream does bi-directional stream forwarding.
// Returns a gRPC status-compatible error.
func (r *router) forwardStream(log *zap.Logger, kasStream grpc.ClientStream, stream grpc.ServerStream) error {
	// Cancellation
	//
	// kasStream is an outbound client stream (this kas -> gateway kas)
	// stream is an inbound server stream (internal/external gRPC client -> this kas)
	//
	// If one of the streams break, the other one needs to be aborted too ASAP. Waiting for a timeout
	// is a waste of resources and a bad API with unpredictable latency.
	//
	// kasStream is automatically aborted if there is a problem with stream because kasStream uses stream's context.
	// Unlike the above, if there is a problem with kasStream, stream.RecvMsg()/stream.SendMsg() are unaffected
	// so can stay blocked for an arbitrary amount of time.
	// To make gRPC abort those method calls, gRPC stream handler (i.e. this method) should just return from the call.
	// See https://github.com/grpc/grpc-go/issues/465#issuecomment-179414474
	// To implement this, we read and write in both directions in separate goroutines and return from this
	// handler whenever there is an error, aborting both connections.

	// Channel of size 1 to ensure that if we return early, the second goroutine has space for the value.
	// We don't care about the second value if the first one is a non-nil error.
	res := make(chan *status.Status, 1)
	go func() {
		res <- r.pipeFromKasToStream(log, kasStream, stream)
	}()
	go func() {
		res <- pipeFromStreamToKas(kasStream, stream)
	}()
	resStatus := <-res
	if resStatus != nil {
		return resStatus.Err()
	}
	resStatus = <-res
	if resStatus != nil {
		return resStatus.Err()
	}
	return nil
}

func (r *router) pipeFromKasToStream(log *zap.Logger, kasStream grpc.ClientStream, stream grpc.ServerStream) *status.Status {
	err := r.gatewayKasVisitor.Visit(kasStream,
		grpctool.WithStartState(tunnelReadyFieldNumber),
		grpctool.WithCallback(tunnelReadyFieldNumber, func(tunnelReady *GatewayKasResponse_TunnelReady) error {
			// It's been read already, shouldn't be received again
			return status.Errorf(codes.InvalidArgument, "Unexpected %T message received", tunnelReady)
		}),
		grpctool.WithCallback(headerFieldNumber, func(header *GatewayKasResponse_Header) error {
			return stream.SetHeader(header.Metadata())
		}),
		grpctool.WithCallback(messageFieldNumber, func(message *GatewayKasResponse_Message) error {
			return stream.SendMsg(&grpctool.RawFrame{
				Data: message.Data,
			})
		}),
		grpctool.WithCallback(trailerFieldNumber, func(trailer *GatewayKasResponse_Trailer) error {
			stream.SetTrailer(trailer.Metadata())
			return nil
		}),
	)
	switch {
	case err == nil:
		return nil
	case grpctool.IsStatusError(err):
		// A gRPC status already
		return status.Convert(err)
	default:
		// Something unexpected
		// TODO track error
		log.Error("Failed to route request: visitor", zap.Error(err))
		return status.New(codes.Unavailable, "unavailable")
	}
}

// pipeFromStreamToKas pipes data kasStream -> stream
// must return gRPC status compatible error or nil.
func pipeFromStreamToKas(kasStream grpc.ClientStream, stream grpc.ServerStream) *status.Status {
	var frame grpctool.RawFrame
	for {
		err := stream.RecvMsg(&frame)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return status.Convert(err) // as is
		}
		err = kasStream.SendMsg(&frame)
		if err != nil {
			return status.Convert(err) // as is
		}
	}
	err := kasStream.CloseSend()
	return status.Convert(err) // as is or nil
}

type tunnelInfoCollector []*tracker.TunnelInfo

func (c *tunnelInfoCollector) Collect(info *tracker.TunnelInfo) (bool, error) {
	if info.KasUrl == "" {
		// kas without a private API endpoint. Ignore it.
		return false, nil
	}
	*c = append(*c, info)
	return false, nil
}
