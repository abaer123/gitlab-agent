package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/retry"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	requestInfoNumber protoreflect.FieldNumber = 1
	messageNumber     protoreflect.FieldNumber = 2
	closeSendNumber   protoreflect.FieldNumber = 3
)

var (
	proxyStreamDesc = grpc.StreamDesc{
		ServerStreams: true,
		ClientStreams: true,
	}
)

type connection struct {
	log                *zap.Logger
	client             rpc.ReverseTunnelClient
	internalServerConn grpc.ClientConnInterface
	streamVisitor      *grpctool.StreamVisitor
	connectRetryPeriod time.Duration
}

func (c *connection) Run(ctx context.Context) {
	retry.JitterUntil(ctx, c.connectRetryPeriod, c.attemptLoop)
}

func (c *connection) attemptLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		err := c.attempt(ctx)
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				c.log.Error("Reverse tunnel", zap.Error(err))
			}
			return
		}
		// successfully handled a connection, re-establish it immediately
	}
}

func (c *connection) attempt(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	var (
		tunnel       rpc.ReverseTunnel_ConnectClient
		clientStream grpc.ClientStream
	)
	clientStreamReady := make(chan struct{})

	// agentk -> kas reverse tunnel
	g.Go(func() error {
		var err error
		tunnel, err = c.client.Connect(ctx)
		if err != nil {
			return fmt.Errorf("Connect(): %w", err) // wrap
		}
		err = tunnel.Send(&rpc.ConnectRequest{
			Msg: &rpc.ConnectRequest_Descriptor_{
				Descriptor_: &rpc.AgentDescriptor{},
			},
		})
		if err != nil {
			return fmt.Errorf("Send(descriptor): %w", err) // wrap
		}
		return c.streamVisitor.Visit(tunnel,
			grpctool.WithCallback(requestInfoNumber, func(reqInfo *rpc.RequestInfo) error {
				outgoingCtx := metadata.NewOutgoingContext(ctx, reqInfo.Metadata())
				clientStream, err = c.internalServerConn.NewStream(outgoingCtx, &proxyStreamDesc, reqInfo.MethodName)
				if err != nil {
					return fmt.Errorf("NewStream(): %w", err)
				}
				close(clientStreamReady) // signal the other goroutine that clientStream var has been set and can be used
				return nil
			}),
			grpctool.WithCallback(messageNumber, func(message *rpc.Message) error {
				err = clientStream.SendMsg(&grpctool.RawFrame{
					Data: message.Data,
				})
				if err != nil {
					return fmt.Errorf("SendMsg(): %w", err)
				}
				return nil
			}),
			grpctool.WithCallback(closeSendNumber, func(closeSend *rpc.CloseSend) error {
				err = clientStream.CloseSend()
				if err != nil {
					return fmt.Errorf("CloseSend(): %w", err)
				}
				return nil
			}),
		)
	})
	// tunnel -> internal server
	g.Go(func() error {
		select {
		case <-ctx.Done():
			return nil
		case <-clientStreamReady:
		}
		header, err := clientStream.Header()
		if err != nil {
			return fmt.Errorf("Header(): %w", err) // wrap
		}
		err = tunnel.Send(&rpc.ConnectRequest{
			Msg: &rpc.ConnectRequest_Header{
				Header: &rpc.Header{
					Meta: rpc.MetaToValuesMap(header),
				},
			},
		})
		if err != nil {
			return fmt.Errorf("Send(header): %w", err) // wrap
		}
		for {
			var frame grpctool.RawFrame
			err = clientStream.RecvMsg(&frame)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return fmt.Errorf("RecvMsg(): %w", err) // wrap
			}
			err = tunnel.Send(&rpc.ConnectRequest{
				Msg: &rpc.ConnectRequest_Message{
					Message: &rpc.Message{
						Data: frame.Data,
					},
				},
			})
			if err != nil {
				return fmt.Errorf("Send(message): %w", err) // wrap
			}
		}
		err = tunnel.Send(&rpc.ConnectRequest{
			Msg: &rpc.ConnectRequest_Trailer{
				Trailer: &rpc.Trailer{
					Meta: rpc.MetaToValuesMap(clientStream.Trailer()),
				},
			},
		})
		if err != nil {
			return fmt.Errorf("Send(trailer): %w", err) // wrap
		}
		err = tunnel.CloseSend()
		if err != nil {
			return fmt.Errorf("CloseSend(): %w", err) // wrap
		}
		return nil
	})
	return g.Wait()
}
