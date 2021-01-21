package reverse_tunnel

import (
	"errors"
	"io"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	agentDescriptorNumber protoreflect.FieldNumber = 1
	headerNumber          protoreflect.FieldNumber = 2
	messageNumber         protoreflect.FieldNumber = 3
	trailerNumber         protoreflect.FieldNumber = 4
	errorNumber           protoreflect.FieldNumber = 5
)

type connection struct {
	tunnel              rpc.ReverseTunnel_ConnectServer
	tunnelStreamVisitor *grpctool.StreamVisitor
	tunnelRetErr        chan<- error
	agentId             int64
	agentProjectId      int64
	agentName           string
	agentDescriptor     *rpc.AgentDescriptor
}

// ForwardStream performs bi-directional message forwarding between incomingStream and c.tunnel.
func (c *connection) ForwardStream(incomingStream grpc.ServerStream) error {
	// Here we have a situation where we need to pipe one server stream into another server stream.
	// One stream is incoming request stream and the other one is incoming tunnel connection stream.
	// We need to use at least one extra goroutine in addition to the current one (or two separate ones) to
	// implement full duplex bidirectional stream piping. One goroutine reads and writes in one direction and the other
	// one in the opposite direction.
	// What if one of them returns an error? We need to unblock the other one, ideally ASAP, to release resources. If
	// it's not unblocked, it'll sit there until it hits a timeout or is aborted by peer. Ok-ish, but far from ideal.
	// To abort request processing on the server side, gRPC stream handler should just return from the call.
	// See https://github.com/grpc/grpc-go/issues/465#issuecomment-179414474
	// To implement this, we read and write in both directions in separate goroutines and return from both
	// handlers whenever there is an error, aborting both connections:
	// - Returning from this function means returning from the incoming request handler.
	// - Sending to c.tunnelRetErr leads to returning that value from the tunnel connection handler.

	// Channel of size 1 to ensure that if we return early, the second goroutine has space for the value.
	// We don't care about the second value if the first one has at least one non-nil error.
	res := make(chan errPair, 1)
	startReadingTunnel := make(chan struct{})
	// Pipe incoming stream (i.e. data a client is sending us) into the tunnel stream
	goErrPair(res, func() (error /* forTunnel */, error /* forIncomingStream */) {
		incomingCtx := incomingStream.Context()
		md, _ := metadata.FromIncomingContext(incomingCtx)
		//if md != nil {
		// TODO sanitize?
		//} else {
		//md = metadata.MD{}
		//}
		err := c.tunnel.Send(&rpc.ConnectResponse{
			Msg: &rpc.ConnectResponse_RequestInfo{
				RequestInfo: &rpc.RequestInfo{
					MethodName: grpc.ServerTransportStreamFromContext(incomingCtx).Method(),
					Meta:       rpc.MetaToValuesMap(md),
				},
			},
		})
		if err != nil {
			return err, err
		}
		close(startReadingTunnel)
		for {
			var frame grpctool.RawFrame
			err = incomingStream.RecvMsg(&frame)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return status.Error(codes.Unavailable, "unavailable"), err
			}
			err = c.tunnel.Send(&rpc.ConnectResponse{
				Msg: &rpc.ConnectResponse_Message{
					Message: &rpc.Message{
						Data: frame.Data,
					},
				},
			})
			if err != nil {
				return err, err
			}
		}
		err = c.tunnel.Send(&rpc.ConnectResponse{
			Msg: &rpc.ConnectResponse_CloseSend{
				CloseSend: &rpc.CloseSend{},
			},
		})
		if err != nil {
			return err, err
		}
		return nil, nil
	})
	// Pipe tunnel stream (i.e. data agentk is sending us) into the incoming stream
	goErrPair(res, func() (error /* forTunnel */, error /* forIncomingStream */) {
		select {
		case <-incomingStream.Context().Done():
			return nil, status.Error(codes.Unavailable, "unavailable")
		case <-startReadingTunnel:
		}
		var forTunnel, forIncomingStream error
		fromVisitor := c.tunnelStreamVisitor.Visit(c.tunnel,
			grpctool.WithStartState(agentDescriptorNumber),
			grpctool.WithCallback(agentDescriptorNumber, func(agentDescriptor *rpc.AgentDescriptor) error {
				// It's been read already, shouldn't be received again
				forTunnel = status.Errorf(codes.InvalidArgument, "Unexpected %T message received", agentDescriptor)
				return status.Errorf(codes.Internal, "Unexpected %T message received", agentDescriptor)
			}),
			grpctool.WithCallback(headerNumber, func(header *rpc.Header) error {
				return incomingStream.SetHeader(header.Metadata())
			}),
			grpctool.WithCallback(messageNumber, func(message *rpc.Message) error {
				return incomingStream.SendMsg(&grpctool.RawFrame{
					Data: message.Data,
				})
			}),
			grpctool.WithCallback(trailerNumber, func(trailer *rpc.Trailer) error {
				incomingStream.SetTrailer(trailer.Metadata())
				return nil
			}),
			grpctool.WithCallback(errorNumber, func(rpcError *rpc.Error) error {
				forIncomingStream = status.ErrorProto(rpcError.Status)
				// Not returning an error since we must be reading from the tunnel stream till io.EOF
				// to properly consume it. There is no need to abort it in this scenario.
				// The server is expected to close the stream (i.e. we'll get io.EOF) right after we got this message.
				return nil
			}),
		)
		if fromVisitor != nil {
			if forIncomingStream == nil {
				forIncomingStream = fromVisitor
			}
			if forTunnel == nil {
				forTunnel = status.Error(codes.Unavailable, "unavailable")
			}
		}
		return forTunnel, forIncomingStream
	})
	pair := <-res
	if !pair.isNil() {
		c.tunnelRetErr <- pair.forTunnel
		return pair.forIncomingStream
	}
	pair = <-res
	c.tunnelRetErr <- pair.forTunnel
	return pair.forIncomingStream
}

type errPair struct {
	forTunnel         error
	forIncomingStream error
}

func (p errPair) isNil() bool {
	return p.forTunnel == nil && p.forIncomingStream == nil
}

func goErrPair(c chan<- errPair, f func() (error /* forTunnel */, error /* forIncomingStream */)) {
	go func() {
		var pair errPair
		pair.forTunnel, pair.forIncomingStream = f()
		c <- pair
	}()
}
