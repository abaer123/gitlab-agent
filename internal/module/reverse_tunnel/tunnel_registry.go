package reverse_tunnel

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel/tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/mathz"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TunnelHandler interface {
	// HandleTunnel is called with server-side interface of the reverse tunnel.
	// It registers the tunnel and blocks, waiting for a request to proxy through the tunnel.
	// The method returns the error value to return to gRPC framework.
	// ctx can be used to unblock the method if the tunnel is not being used already.
	// ctx should be a child of the server's context.
	HandleTunnel(ctx context.Context, agentInfo *api.AgentInfo, server rpc.ReverseTunnel_ConnectServer) error
}

type TunnelFinder interface {
	// FindTunnel finds a tunnel to a matching agentk.
	// It waits for a matching tunnel to proxy a connection through. When a matching tunnel is found, it is returned.
	// It only returns errors from the context or context.Canceled if the finder is shutting down.
	FindTunnel(ctx context.Context, agentId int64) (Tunnel, error)
}

type findTunnelRequest struct {
	agentId int64
	retTun  chan<- Tunnel
}

type TunnelRegistry struct {
	log                   *zap.Logger
	tunnelRegisterer      tracker.Registerer
	ownPrivateApiUrl      string
	tunnelStreamVisitor   *grpctool.StreamVisitor
	tuns                  map[*tunnel]struct{}
	tunsByAgentId         map[int64]map[*tunnel]struct{}
	findRequestsByAgentId map[int64]map[*findTunnelRequest]struct{}

	tunnelRegister   chan *tunnel
	tunnelUnregister chan *tunnel

	findRequest      chan *findTunnelRequest
	findRequestAbort chan *findTunnelRequest
}

func NewTunnelRegistry(log *zap.Logger, tunnelRegisterer tracker.Registerer, ownPrivateApiUrl string) (*TunnelRegistry, error) {
	tunnelStreamVisitor, err := grpctool.NewStreamVisitor(&rpc.ConnectRequest{})
	if err != nil {
		return nil, err
	}
	return &TunnelRegistry{
		log:                   log,
		tunnelRegisterer:      tunnelRegisterer,
		ownPrivateApiUrl:      ownPrivateApiUrl,
		tunnelStreamVisitor:   tunnelStreamVisitor,
		tuns:                  make(map[*tunnel]struct{}),
		tunsByAgentId:         make(map[int64]map[*tunnel]struct{}),
		findRequestsByAgentId: make(map[int64]map[*findTunnelRequest]struct{}),

		tunnelRegister:   make(chan *tunnel),
		tunnelUnregister: make(chan *tunnel),

		findRequest:      make(chan *findTunnelRequest),
		findRequestAbort: make(chan *findTunnelRequest),
	}, nil
}

func (r *TunnelRegistry) Run(ctx context.Context) error {
	defer r.cleanup()
	for {
		select {
		case <-ctx.Done():
			return nil
		case toReg := <-r.tunnelRegister:
			r.handleTunnelRegister(toReg)
		case toUnreg := <-r.tunnelUnregister:
			r.handleTunnelUnregister(toUnreg)
		case ftr := <-r.findRequest:
			r.handleFindRequest(ftr)
		case ftr := <-r.findRequestAbort:
			r.handleFindRequestAbort(ftr)
		}
	}
}

func (r *TunnelRegistry) FindTunnel(ctx context.Context, agentId int64) (Tunnel, error) {
	retTun := make(chan Tunnel) // can receive nil from it if registry is shutting down
	ftr := &findTunnelRequest{
		agentId: agentId,
		retTun:  retTun,
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r.findRequest <- ftr:
	}
	select {
	case <-ctx.Done():
		select {
		case r.findRequestAbort <- ftr:
		case tun := <-retTun:
			if tun != nil {
				// Got the tunnel, but it's too late. Must just close it as it's impossible to register it again.
				// The agent will immediately reconnect so not a big deal.
				tun.Done()
			}
		}
		return nil, ctx.Err()
	case tun := <-retTun:
		if tun == nil {
			return nil, context.Canceled
		}
		return tun, nil
	}
}

func (r *TunnelRegistry) HandleTunnel(ctx context.Context, agentInfo *api.AgentInfo, server rpc.ReverseTunnel_ConnectServer) error {
	recv, err := server.Recv()
	if err != nil {
		if !grpctool.RequestCanceled(err) {
			r.log.Debug("Recv() from incoming tunnel connection failed", logz.Error(err))
		}
		return status.Error(codes.Unavailable, "unavailable")
	}
	descriptor, ok := recv.Msg.(*rpc.ConnectRequest_Descriptor_)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "Invalid oneof value type: %T", recv.Msg)
	}
	retErr := make(chan error)
	tun := &tunnel{
		tunnel:              server,
		tunnelStreamVisitor: r.tunnelStreamVisitor,
		tunnelRetErr:        retErr,
		tunnelInfo: &tracker.TunnelInfo{
			AgentDescriptor: descriptor.Descriptor_.AgentDescriptor,
			ConnectionId:    mathz.Int63(),
			AgentId:         agentInfo.Id,
			KasUrl:          r.ownPrivateApiUrl,
		},
	}
	// Register
	select {
	case <-ctx.Done():
		// Stream canceled - nothing to do, just close the stream
		return nil
	case r.tunnelRegister <- tun:
		// Successfully registered the stream with the main goroutine
	}
	// Wait for return error or for cancellation
	select {
	case <-ctx.Done():
		// Context canceled
		select {
		case r.tunnelUnregister <- tun: // let the main goroutine know
			// main or tunnel-using goroutine must return an error value to return
			// it must either be handleTunnelUnregister() or the tunnel.ForwardStream().
			return <-retErr
		case err = <-retErr: // value to return is available already
			return err
		}
	case err = <-retErr:
		return err
	}
}

func (r *TunnelRegistry) handleTunnelRegister(toReg *tunnel) {
	// 1. Before registering the tunnel see if there is a find tunnel request waiting for it
	findRequestsForAgentId := r.findRequestsByAgentId[toReg.tunnelInfo.AgentId]
	for ftr := range findRequestsForAgentId {
		// Waiting request found!
		ftr.retTun <- toReg      // Satisfy the waiting request ASAP
		r.deleteFindRequest(ftr) // Remove it from the queue
		return
	}

	// 2. Register the tunnel
	r.tunnelRegisterer.RegisterTunnel(context.Background(), toReg.tunnelInfo) // register ASAP
	r.tuns[toReg] = struct{}{}
	tunsByAgentId := r.tunsByAgentId[toReg.tunnelInfo.AgentId]
	if tunsByAgentId == nil {
		tunsByAgentId = make(map[*tunnel]struct{}, 1)
		r.tunsByAgentId[toReg.tunnelInfo.AgentId] = tunsByAgentId
	}
	tunsByAgentId[toReg] = struct{}{}
}

func (r *TunnelRegistry) handleTunnelUnregister(toUnreg *tunnel) {
	if r.tunsByAgentId[toUnreg.tunnelInfo.AgentId] != nil { // Tunnel might not be there if it's been obtained from the map already
		r.unregisterTunnel(toUnreg)
		toUnreg.tunnelRetErr <- nil
	}
}

func (r *TunnelRegistry) unregisterTunnel(toAbort *tunnel) {
	r.tunnelRegisterer.UnregisterTunnel(context.Background(), toAbort.tunnelInfo)
	delete(r.tuns, toAbort)
	tunsByAgentId := r.tunsByAgentId[toAbort.tunnelInfo.AgentId]
	delete(tunsByAgentId, toAbort)
	if len(tunsByAgentId) == 0 {
		delete(r.tunsByAgentId, toAbort.tunnelInfo.AgentId)
	}
}

func (r *TunnelRegistry) handleFindRequest(ftr *findTunnelRequest) {
	// 1. Check if we have a suitable tunnel
	for tun := range r.tunsByAgentId[ftr.agentId] {
		// Suitable tunnel found!
		ftr.retTun <- tun // respond ASAP, then do all the bookkeeping
		r.unregisterTunnel(tun)
		return
	}

	// 2. No suitable tunnel found, add to the queue
	findRequestsForAgentId := r.findRequestsByAgentId[ftr.agentId]
	if findRequestsForAgentId == nil {
		findRequestsForAgentId = make(map[*findTunnelRequest]struct{}, 1)
		r.findRequestsByAgentId[ftr.agentId] = findRequestsForAgentId
	}
	findRequestsForAgentId[ftr] = struct{}{}
}

func (r *TunnelRegistry) handleFindRequestAbort(ftr *findTunnelRequest) {
	r.deleteFindRequest(ftr)
}

func (r *TunnelRegistry) deleteFindRequest(ftr *findTunnelRequest) {
	findRequestsForAgentId := r.findRequestsByAgentId[ftr.agentId]
	delete(findRequestsForAgentId, ftr)
	if len(findRequestsForAgentId) == 0 {
		delete(r.findRequestsByAgentId, ftr.agentId)
	}
}

func (r *TunnelRegistry) cleanup() {
	// Abort all tunnels
	for c := range r.tuns {
		r.handleTunnelUnregister(c)
	}
	// Abort all waiting new stream requests
	for _, findRequestsForAgentId := range r.findRequestsByAgentId {
		for ftr := range findRequestsForAgentId {
			ftr.retTun <- nil // respond ASAP, then do all the bookkeeping
			r.deleteFindRequest(ftr)
		}
	}
}
