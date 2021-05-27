package kasapp

import (
	"errors"
	"io"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/prototool"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (r *router) RouteToCorrectAgentHandler(srv interface{}, stream grpc.ServerStream) error {
	md, _ := metadata.FromIncomingContext(stream.Context())
	agentId, err := agentIdFromMeta(md)
	if err != nil {
		return err
	}
	wrappedStream := grpc_middleware.WrapServerStream(stream)
	// Overwrite incoming MD with sanitized MD
	wrappedStream.WrappedContext = metadata.NewIncomingContext(
		wrappedStream.WrappedContext,
		removeHopMeta(md),
	)
	stream = wrappedStream
	tunnel, err := r.tunnelFinder.FindTunnel(wrappedStream.WrappedContext, agentId)
	if err != nil {
		return status.FromContextError(err).Err()
	}
	defer tunnel.Done()
	err = stream.SendMsg(&GatewayKasResponse{
		Msg: &GatewayKasResponse_TunnelReady_{
			TunnelReady: &GatewayKasResponse_TunnelReady{},
		},
	})
	if err != nil {
		return err
	}
	var start StartStreaming
	err = stream.RecvMsg(&start)
	if err != nil {
		if errors.Is(err, io.EOF) {
			// Routing kas decided not to proceed
			return nil
		}
		return err
	}
	return tunnel.ForwardStream(stream, wrappingCallback{stream: stream})
}

func removeHopMeta(md metadata.MD) metadata.MD {
	md = md.Copy()
	for k := range md {
		if strings.HasPrefix(k, modserver.RoutingHopPrefix) {
			delete(md, k)
		}
	}
	return md
}

var (
	_ reverse_tunnel.TunnelDataCallback = wrappingCallback{}
)

type wrappingCallback struct {
	stream grpc.ServerStream
}

func (c wrappingCallback) Header(md map[string]*prototool.Values) error {
	return c.stream.SendMsg(&GatewayKasResponse{
		Msg: &GatewayKasResponse_Header_{
			Header: &GatewayKasResponse_Header{
				Meta: md,
			},
		},
	})
}

func (c wrappingCallback) Message(data []byte) error {
	return c.stream.SendMsg(&GatewayKasResponse{
		Msg: &GatewayKasResponse_Message_{
			Message: &GatewayKasResponse_Message{
				Data: data,
			},
		},
	})
}

func (c wrappingCallback) Trailer(md map[string]*prototool.Values) error {
	return c.stream.SendMsg(&GatewayKasResponse{
		Msg: &GatewayKasResponse_Trailer_{
			Trailer: &GatewayKasResponse_Trailer{
				Meta: md,
			},
		},
	})
}

func (c wrappingCallback) Error(stat *statuspb.Status) error {
	return c.stream.SendMsg(&GatewayKasResponse{
		Msg: &GatewayKasResponse_Error_{
			Error: &GatewayKasResponse_Error{
				Status: stat,
			},
		},
	})
}
