package kasapp

import (
	"strconv"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type kasRouter interface {
	RegisterAgentApi(desc *grpc.ServiceDesc)
}

// router is quite dumb at the moment, but we'll make it much better in the next iterations:
// - potentially maintain a short lived cache of per-agent id tunnel info so than subsequent requests can
//   avoid making a tracker.Querier lookup (i.e. Redis lookup)
// - if no tunnels are available for an agent id, block and wait for a notification from
//   the tracker.Querier, which can use a pub-sub for notifications about new tunnels.
// - connections to kas are single use only. They should be cached for reuse and removed if the destination kas
//   goes away. This should be achieved by pub/sub notifications on kas come/go events and/or periodic relisting
//   of currently registered kas instances.
//
// Optimizations above would reduce worst case latency quite a bit.

// router routes traffic from kas to another kas to agentk.
// routing kas -> gateway kas -> agentk
type router struct {
	kasPool       KasPool
	tunnelQuerier tracker.Querier
	tunnelFinder  reverse_tunnel.TunnelFinder
	// internalServer is the internal gRPC server for use inside of kas.
	// Request handlers can obtain the per-request logger using grpctool.LoggerFromContext(requestContext).
	internalServer grpc.ServiceRegistrar
	// privateApiServer is the gRPC server that other kas instances can talk to.
	// Request handlers can obtain the per-request logger using grpctool.LoggerFromContext(requestContext).
	privateApiServer  grpc.ServiceRegistrar
	gatewayKasVisitor *grpctool.StreamVisitor
}

func (r *router) RegisterAgentApi(desc *grpc.ServiceDesc) {
	// 1. Munge the descriptor into the right shape:
	//    - turn all unary calls into streaming calls
	//    - all streaming calls, including the ones from above, are handled by routing handlers
	internalServerDesc := mungeDescriptor(desc, r.RouteToCorrectKasHandler)
	privateApiServerDesc := mungeDescriptor(desc, r.RouteToCorrectAgentHandler)

	// 2. Register on InternalServer gRPC server so that ReverseTunnelClient can be used in kas to send data to
	//    this API within this kas instance. This kas instance then routes the stream to the gateway kas instance.
	r.internalServer.RegisterService(internalServerDesc, nil)

	// 3. Register on PrivateApiServer gRPC server so that this kas instance can act as the gateway kas instance
	//    from above and then route to one of the matching connected agentk instances.
	r.privateApiServer.RegisterService(privateApiServerDesc, nil)
}

func mungeDescriptor(in *grpc.ServiceDesc, handler grpc.StreamHandler) *grpc.ServiceDesc {
	streams := make([]grpc.StreamDesc, 0, len(in.Streams)+len(in.Methods))
	for _, stream := range in.Streams {
		streams = append(streams, grpc.StreamDesc{
			StreamName:    stream.StreamName,
			Handler:       handler,
			ServerStreams: true,
			ClientStreams: true,
		})
	}
	// Turn all methods into streams
	for _, method := range in.Methods {
		streams = append(streams, grpc.StreamDesc{
			StreamName:    method.MethodName,
			Handler:       handler,
			ServerStreams: true,
			ClientStreams: true,
		})
	}
	return &grpc.ServiceDesc{
		ServiceName: in.ServiceName,
		Streams:     streams,
		Metadata:    in.Metadata,
	}
}

func agentIdFromMeta(md metadata.MD) (int64, error) {
	val := md.Get(modserver.RoutingAgentIdMetadataKey)
	if len(val) != 1 {
		return 0, status.Errorf(codes.InvalidArgument, "Expecting a single %s, got %d", modserver.RoutingAgentIdMetadataKey, len(val))
	}
	agentId, err := strconv.ParseInt(val[0], 10, 64)
	if err != nil {
		return 0, status.Errorf(codes.InvalidArgument, "Invalid %s", modserver.RoutingAgentIdMetadataKey)
	}
	return agentId, nil
}
