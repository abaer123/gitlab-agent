package reverse_tunnel

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TunnelConnectionHandler interface {
	// HandleTunnelConnection is called with server-side interface of the reverse tunnel.
	// It registers the connection and blocks, waiting for a request to proxy through the connection.
	// The method returns the error value to return to gRPC framework.
	// ctx can be used to unblock the method if the connection is not being used already.
	// ctx should be a child of the server's context.
	HandleTunnelConnection(ctx context.Context, agentInfo *api.AgentInfo, server rpc.ReverseTunnel_ConnectServer) error
}

type IncomingConnectionHandler interface {
	// HandleIncomingConnection is called with server-side interface of the incoming connection.
	// It registers the connection and blocks, waiting for a matching tunnel to proxy the connection through.
	// The method returns the error value to return to gRPC framework.
	HandleIncomingConnection(agentId int64, stream grpc.ServerStream) error
}

type connForwardRequest struct {
	agentId int64
	retConn chan<- *connection
}

type ConnectionRegistry struct {
	log                   *zap.Logger
	tunnelStreamVisitor   *grpctool.StreamVisitor
	conns                 map[*connection]struct{}
	connsByAgentId        map[int64]map[*connection]struct{}
	connRequestsByAgentId map[int64]map[*connForwardRequest]struct{}

	connRegister   chan *connection
	connUnregister chan *connection

	connRequest      chan *connForwardRequest
	connRequestAbort chan *connForwardRequest
}

func NewConnectionRegistry(log *zap.Logger) (*ConnectionRegistry, error) {
	tunnelStreamVisitor, err := grpctool.NewStreamVisitor(&rpc.ConnectRequest{})
	if err != nil {
		return nil, err
	}
	return &ConnectionRegistry{
		log:                   log,
		tunnelStreamVisitor:   tunnelStreamVisitor,
		conns:                 make(map[*connection]struct{}),
		connsByAgentId:        make(map[int64]map[*connection]struct{}),
		connRequestsByAgentId: make(map[int64]map[*connForwardRequest]struct{}),

		connRegister:   make(chan *connection),
		connUnregister: make(chan *connection),

		connRequest:      make(chan *connForwardRequest),
		connRequestAbort: make(chan *connForwardRequest),
	}, nil
}

func (r *ConnectionRegistry) Run(ctx context.Context) error {
	defer r.cleanup()
	for {
		select {
		case <-ctx.Done():
			return nil
		case toReg := <-r.connRegister:
			r.handleConnRegister(toReg)
		case toUnreg := <-r.connUnregister:
			r.handleConnUnregister(toUnreg)
		case connRequest := <-r.connRequest:
			r.handleConnRequest(connRequest)
		case connRequestAbort := <-r.connRequestAbort:
			r.handleConnRequestAbort(connRequestAbort)
		}
	}
}

func (r *ConnectionRegistry) HandleIncomingConnection(agentId int64, stream grpc.ServerStream) error {
	retConn := make(chan *connection) // can receive nil from it if registry is shutting down
	s := &connForwardRequest{
		agentId: agentId,
		retConn: retConn,
	}
	ctx := stream.Context()
	select {
	case <-ctx.Done():
		return status.Error(codes.Canceled, "context done")
	case r.connRequest <- s:
	}
	select {
	case <-ctx.Done():
		select {
		case r.connRequestAbort <- s:
		case conn := <-retConn:
			if conn != nil {
				// Got the connection, but it's too late. Must just close it as it's impossible to register it again.
				// The agent will immediately reconnect so not a big deal.
				conn.tunnelRetErr <- nil
			}
		}
		return status.Error(codes.Canceled, "context done")
	case conn := <-retConn:
		if conn == nil {
			return status.Error(codes.Unavailable, "unavailable")
		}
		return conn.ForwardStream(stream)
	}
}

func (r *ConnectionRegistry) HandleTunnelConnection(ctx context.Context, agentInfo *api.AgentInfo, server rpc.ReverseTunnel_ConnectServer) error {
	recv, err := server.Recv()
	if err != nil {
		if !grpctool.RequestCanceled(err) {
			r.log.Debug("Recv() from incoming tunnel connection failed", zap.Error(err))
		}
		return status.Error(codes.Unavailable, "unavailable")
	}
	descriptor, ok := recv.Msg.(*rpc.ConnectRequest_Descriptor_)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "Invalid oneof value type: %T", recv.Msg)
	}
	retErr := make(chan error)
	c := &connection{
		tunnel:              server,
		tunnelStreamVisitor: r.tunnelStreamVisitor,
		tunnelRetErr:        retErr,
		agentId:             agentInfo.Id,
		agentDescriptor:     descriptor.Descriptor_.AgentDescriptor,
	}
	// Register
	select {
	case <-ctx.Done():
		// Stream canceled - nothing to do, just close the stream
		return nil
	case r.connRegister <- c:
		// Successfully registered the stream with the main goroutine
	}
	// Wait for return error or for cancellation
	select {
	case <-ctx.Done():
		// Context canceled
		select {
		case r.connUnregister <- c: // let the main goroutine know
			// main or connection-using goroutine must return an error value to return
			// it must either be handleConnUnregister() or the connection.ForwardStream().
			return <-retErr
		case err = <-retErr: // value to return is available already
			return err
		}
	case err = <-retErr:
		return err
	}
}

func (r *ConnectionRegistry) handleConnRegister(toReg *connection) {
	// 1. Before registering the connection see if there is a connection request waiting for it
	connRequestsForAgentId := r.connRequestsByAgentId[toReg.agentId]
	for connReq := range connRequestsForAgentId {
		// Waiting request found!
		r.deleteConnRequest(connReq) // Remove it from the queue
		connReq.retConn <- toReg     // Satisfy the waiting request
		return
	}

	// 2. Register the connection
	r.conns[toReg] = struct{}{}
	connsByAgentId := r.connsByAgentId[toReg.agentId]
	if connsByAgentId == nil {
		connsByAgentId = make(map[*connection]struct{}, 1)
		r.connsByAgentId[toReg.agentId] = connsByAgentId
	}
	connsByAgentId[toReg] = struct{}{}
}

func (r *ConnectionRegistry) handleConnUnregister(toUnreg *connection) {
	if r.connsByAgentId[toUnreg.agentId] != nil { // Connection might not be there if it's been obtained from the map already
		r.unregisterConnection(toUnreg)
		toUnreg.tunnelRetErr <- status.Error(codes.Canceled, "context done")
	}
}

func (r *ConnectionRegistry) unregisterConnection(toAbort *connection) {
	delete(r.conns, toAbort)
	connsForAgentId := r.connsByAgentId[toAbort.agentId]
	delete(connsForAgentId, toAbort)
	if len(connsForAgentId) == 0 {
		delete(r.connsByAgentId, toAbort.agentId)
	}
}

func (r *ConnectionRegistry) handleConnRequest(connRequest *connForwardRequest) {
	// 1. Check if we have a suitable connection
	for conn := range r.connsByAgentId[connRequest.agentId] {
		// Suitable connection found!
		r.unregisterConnection(conn)
		connRequest.retConn <- conn
		return
	}

	// 2. No suitable connection found, add to the queue
	connRequestsForAgentId := r.connRequestsByAgentId[connRequest.agentId]
	if connRequestsForAgentId == nil {
		connRequestsForAgentId = make(map[*connForwardRequest]struct{}, 1)
		r.connRequestsByAgentId[connRequest.agentId] = connRequestsForAgentId
	}
	connRequestsForAgentId[connRequest] = struct{}{}
}

func (r *ConnectionRegistry) handleConnRequestAbort(connRequestAbort *connForwardRequest) {
	r.deleteConnRequest(connRequestAbort)
}

func (r *ConnectionRegistry) deleteConnRequest(connRequestAbort *connForwardRequest) {
	connRequestsForAgentId := r.connRequestsByAgentId[connRequestAbort.agentId]
	delete(connRequestsForAgentId, connRequestAbort)
	if len(connRequestsForAgentId) == 0 {
		delete(r.connRequestsByAgentId, connRequestAbort.agentId)
	}
}

func (r *ConnectionRegistry) cleanup() {
	// Abort all connections
	for c := range r.conns {
		r.handleConnUnregister(c)
	}
	// Abort all waiting new stream requests
	for _, connRequestsForAgentId := range r.connRequestsByAgentId {
		for connReq := range connRequestsForAgentId {
			r.deleteConnRequest(connReq)
			connReq.retConn <- nil
		}
	}
}
