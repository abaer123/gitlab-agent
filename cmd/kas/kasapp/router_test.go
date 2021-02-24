package kasapp

import (
	"context"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/info"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool/test"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_reverse_tunnel_tracker"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	agentId int64 = 234
)

var (
	_ kasRouter          = &router{}
	_ grpc.StreamHandler = (&router{}).RouteToCorrectKasHandler
	_ grpc.StreamHandler = (&router{}).RouteToCorrectAgentHandler
)

func TestRouter_UnaryHappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	unaryResponse := &test.Response{Message: &test.Response_Scalar{Scalar: 123}}
	routingMetadata := modserver.RoutingMetadata(agentId)
	payloadMD := metadata.Pairs("key1", "value1")
	payloadReq := &test.Request{S1: "123"}
	responseMD := metadata.Pairs("key2", "value2")
	trailersMD := metadata.Pairs("key3", "value3")
	var (
		headerResp  metadata.MD
		trailerResp metadata.MD
	)
	tunnel := mock_reverse_tunnel.NewMockTunnel(ctrl)
	tunnelForwardStream := tunnel.EXPECT().
		ForwardStream(gomock.Any(), gomock.Any()).
		Do(forwardStream(t, routingMetadata, payloadMD, payloadReq, unaryResponse, responseMD, trailersMD))
	runRouterTest(t, tunnel, tunnelForwardStream, func(client test.TestingClient) {
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Join(routingMetadata, payloadMD))
		response, err := client.RequestResponse(ctx, payloadReq, grpc.Header(&headerResp), grpc.Trailer(&trailerResp))
		require.NoError(t, err)
		assert.Empty(t, cmp.Diff(response, unaryResponse, protocmp.Transform()))
		mdContains(t, responseMD, headerResp)
		mdContains(t, trailersMD, trailerResp)
	})
}

func TestRouter_UnaryImmediateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	routingMetadata := modserver.RoutingMetadata(agentId)
	statusWithDetails, err := status.New(codes.InvalidArgument, "Some expected error").
		WithDetails(&test.Request{S1: "some details of the error"})
	require.NoError(t, err)
	tunnel := mock_reverse_tunnel.NewMockTunnel(ctrl)
	tunnelForwardStream := tunnel.EXPECT().
		ForwardStream(gomock.Any(), gomock.Any()).
		Return(statusWithDetails.Err())
	runRouterTest(t, tunnel, tunnelForwardStream, func(client test.TestingClient) {
		ctx := metadata.NewOutgoingContext(context.Background(), routingMetadata)
		_, err = client.RequestResponse(ctx, &test.Request{S1: "123"})
		require.Error(t, err)
		receivedStatus := status.Convert(err).Proto()
		assert.Empty(t, cmp.Diff(receivedStatus, statusWithDetails.Proto(), protocmp.Transform()))
	})
}

func TestRouter_UnaryErrorAfterHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	routingMetadata := modserver.RoutingMetadata(agentId)
	payloadMD := metadata.Pairs("key1", "value1")
	responseMD := metadata.Pairs("key2", "value2")
	trailersMD := metadata.Pairs("key3", "value3")
	statusWithDetails, err := status.New(codes.InvalidArgument, "Some expected error").
		WithDetails(&test.Request{S1: "some details of the error"})
	require.NoError(t, err)
	var (
		headerResp  metadata.MD
		trailerResp metadata.MD
	)
	tunnel := mock_reverse_tunnel.NewMockTunnel(ctrl)
	tunnelForwardStream := tunnel.EXPECT().
		ForwardStream(gomock.Any(), gomock.Any()).
		DoAndReturn(func(incomingStream grpc.ServerStream, cb reverse_tunnel.TunnelDataCallback) error {
			verifyMeta(t, incomingStream, routingMetadata, payloadMD)
			assert.NoError(t, cb.Header(grpctool.MetaToValuesMap(responseMD)))
			assert.NoError(t, cb.Trailer(grpctool.MetaToValuesMap(trailersMD)))
			return statusWithDetails.Err()
		})
	runRouterTest(t, tunnel, tunnelForwardStream, func(client test.TestingClient) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx = metadata.NewOutgoingContext(ctx, metadata.Join(routingMetadata, payloadMD))
		_, err := client.RequestResponse(ctx, &test.Request{S1: "123"}, grpc.Header(&headerResp), grpc.Trailer(&trailerResp))
		require.Error(t, err)
		receivedStatus := status.Convert(err).Proto()
		assert.Empty(t, cmp.Diff(receivedStatus, statusWithDetails.Proto(), protocmp.Transform()))
		mdContains(t, responseMD, headerResp)
		mdContains(t, trailersMD, trailerResp)
	})
}

func TestRouter_StreamHappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	streamResponse := &test.Response{Message: &test.Response_Scalar{Scalar: 123}}
	routingMetadata := modserver.RoutingMetadata(agentId)
	payloadMD := metadata.Pairs("key1", "value1")
	payloadReq := &test.Request{S1: "123"}
	responseMD := metadata.Pairs("key2", "value2")
	trailersMD := metadata.Pairs("key3", "value3")
	tunnel := mock_reverse_tunnel.NewMockTunnel(ctrl)
	tunnelForwardStream := tunnel.EXPECT().
		ForwardStream(gomock.Any(), gomock.Any()).
		Do(forwardStream(t, routingMetadata, payloadMD, payloadReq, streamResponse, responseMD, trailersMD))
	runRouterTest(t, tunnel, tunnelForwardStream, func(client test.TestingClient) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx = metadata.NewOutgoingContext(ctx, metadata.Join(routingMetadata, payloadMD))
		stream, err := client.StreamingRequestResponse(ctx)
		require.NoError(t, err)
		err = stream.Send(payloadReq)
		require.NoError(t, err)
		err = stream.CloseSend()
		require.NoError(t, err)
		response, err := stream.Recv()
		require.NoError(t, err)
		assert.Empty(t, cmp.Diff(response, streamResponse, protocmp.Transform()))
		_, err = stream.Recv()
		assert.Equal(t, io.EOF, err)
		verifyHeaderAndTrailer(t, stream, responseMD, trailersMD)
	})
}

func TestRouter_StreamImmediateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	routingMetadata := modserver.RoutingMetadata(agentId)
	statusWithDetails, err := status.New(codes.InvalidArgument, "Some expected error").
		WithDetails(&test.Request{S1: "some details of the error"})
	require.NoError(t, err)
	tunnel := mock_reverse_tunnel.NewMockTunnel(ctrl)
	tunnelForwardStream := tunnel.EXPECT().
		ForwardStream(gomock.Any(), gomock.Any()).
		Return(statusWithDetails.Err())
	runRouterTest(t, tunnel, tunnelForwardStream, func(client test.TestingClient) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx = metadata.NewOutgoingContext(ctx, routingMetadata)
		stream, err := client.StreamingRequestResponse(ctx)
		require.NoError(t, err)
		err = stream.Send(&test.Request{S1: "123"})
		require.NoError(t, err)
		err = stream.CloseSend()
		require.NoError(t, err)
		_, err = stream.Recv()
		require.Error(t, err)
		receivedStatus := status.Convert(err).Proto()
		assert.Empty(t, cmp.Diff(receivedStatus, statusWithDetails.Proto(), protocmp.Transform()))
	})
}

func TestRouter_StreamErrorAfterHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	routingMetadata := modserver.RoutingMetadata(agentId)
	payloadMD := metadata.Pairs("key1", "value1")
	responseMD := metadata.Pairs("key2", "value2")
	trailersMD := metadata.Pairs("key3", "value3")
	statusWithDetails, err := status.New(codes.InvalidArgument, "Some expected error").
		WithDetails(&test.Request{S1: "some details of the error"})
	require.NoError(t, err)
	tunnel := mock_reverse_tunnel.NewMockTunnel(ctrl)
	tunnelForwardStream := tunnel.EXPECT().
		ForwardStream(gomock.Any(), gomock.Any()).
		DoAndReturn(func(incomingStream grpc.ServerStream, cb reverse_tunnel.TunnelDataCallback) error {
			verifyMeta(t, incomingStream, routingMetadata, payloadMD)
			assert.NoError(t, cb.Header(grpctool.MetaToValuesMap(responseMD)))
			assert.NoError(t, cb.Trailer(grpctool.MetaToValuesMap(trailersMD)))
			return statusWithDetails.Err()
		})
	runRouterTest(t, tunnel, tunnelForwardStream, func(client test.TestingClient) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx = metadata.NewOutgoingContext(ctx, metadata.Join(routingMetadata, payloadMD))
		stream, err := client.StreamingRequestResponse(ctx)
		require.NoError(t, err)
		err = stream.Send(&test.Request{S1: "123"})
		require.NoError(t, err)
		err = stream.CloseSend()
		require.NoError(t, err)
		_, err = stream.Recv()
		require.Error(t, err)
		receivedStatus := status.Convert(err).Proto()
		assert.Empty(t, cmp.Diff(receivedStatus, statusWithDetails.Proto(), protocmp.Transform()))
		verifyHeaderAndTrailer(t, stream, responseMD, trailersMD)
	})
}

func verifyHeaderAndTrailer(t *testing.T, stream grpc.ClientStream, responseMD, trailersMD metadata.MD) {
	headerResp, err := stream.Header()
	require.NoError(t, err)
	mdContains(t, responseMD, headerResp)
	mdContains(t, trailersMD, stream.Trailer())
}

func forwardStream(t *testing.T, routingMetadata, payloadMD metadata.MD, payloadReq *test.Request, response *test.Response, responseMD, trailersMD metadata.MD) func(grpc.ServerStream, reverse_tunnel.TunnelDataCallback) {
	return func(incomingStream grpc.ServerStream, cb reverse_tunnel.TunnelDataCallback) {
		verifyMeta(t, incomingStream, routingMetadata, payloadMD)
		var req test.Request
		err := incomingStream.RecvMsg(&req)
		assert.NoError(t, err)
		assert.Empty(t, cmp.Diff(payloadReq, &req, protocmp.Transform()))
		data, err := proto.Marshal(response)
		assert.NoError(t, err)
		assert.NoError(t, cb.Header(grpctool.MetaToValuesMap(responseMD)))
		assert.NoError(t, cb.Message(data))
		assert.NoError(t, cb.Trailer(grpctool.MetaToValuesMap(trailersMD)))
	}
}

func verifyMeta(t *testing.T, incomingStream grpc.ServerStream, routingMetadata, payloadMD metadata.MD) {
	md, _ := metadata.FromIncomingContext(incomingStream.Context())
	for k := range routingMetadata { // no routing metadata is passed to the agent
		assert.NotContains(t, md, k)
	}
	mdContains(t, payloadMD, md)
}

func mdContains(t *testing.T, expectedMd metadata.MD, actualMd metadata.MD) {
	for k, v := range expectedMd {
		assert.Equalf(t, v, actualMd[k], "key: %s", k)
	}
}

// test:client(default codec) --> kas1:internal server(raw codec) --> router_kas handler -->
// client from kas_pool(raw wih fallback codec) --> kas2:private server(raw wih fallback codec) -->
// router_agent handler --> tunnel finder --> tunnel.ForwardStream()
func runRouterTest(t *testing.T, tunnel *mock_reverse_tunnel.MockTunnel, tunnelForwardStream *gomock.Call, runTest func(client test.TestingClient)) {
	ctrl := gomock.NewController(t)
	querier := mock_reverse_tunnel_tracker.NewMockQuerier(ctrl)
	finder := mock_reverse_tunnel.NewMockTunnelFinder(ctrl)
	internalServerListener := grpctool.NewDialListener()
	defer internalServerListener.Close()
	privateApiServerListener := grpctool.NewDialListener()
	defer privateApiServerListener.Close()

	gomock.InOrder(
		querier.EXPECT().
			GetTunnelsByAgentId(gomock.Any(), agentId, gomock.Any()).
			DoAndReturn(func(ctx context.Context, agentId int64, cb tracker.GetTunnelsByAgentIdCallback) error {
				done, err := cb(&tracker.TunnelInfo{
					AgentDescriptor: &info.AgentDescriptor{
						Services: []*info.Service{
							{
								Name: "gitlab.agent.grpctool.test.Testing",
								Methods: []*info.Method{
									{
										Name: "RequestResponse",
									},
									{
										Name: "StreamingRequestResponse",
									},
								},
							},
						},
					},
					ConnectionId: 1312312313,
					AgentId:      agentId,
					KasUrl:       "tcp://pipe",
				})
				assert.False(t, done)
				return err
			}),
		finder.EXPECT().
			FindTunnel(gomock.Any(), agentId).
			Return(tunnel, nil),
		tunnelForwardStream,
		tunnel.EXPECT().Done(),
	)

	log := zaptest.NewLogger(t)
	internalServer := grpc.NewServer(
		grpc.ChainStreamInterceptor(
			grpctool.StreamServerCtxAugmentingInterceptor(grpctool.LoggerInjector(log)),
		),
		grpc.ChainUnaryInterceptor(
			grpctool.UnaryServerCtxAugmentingInterceptor(grpctool.LoggerInjector(log)),
		),
		// TODO Stop using the deprecated API once https://github.com/grpc/grpc-go/issues/3694 is resolved
		grpc.CustomCodec(grpctool.RawCodec{}), // nolint: staticcheck
	)
	privateApiServer := grpc.NewServer(
		grpc.ChainStreamInterceptor(
			grpctool.StreamServerCtxAugmentingInterceptor(grpctool.LoggerInjector(log)),
		),
		grpc.ChainUnaryInterceptor(
			grpctool.UnaryServerCtxAugmentingInterceptor(grpctool.LoggerInjector(log)),
		),
		// TODO Stop using the deprecated API once https://github.com/grpc/grpc-go/issues/3694 is resolved
		grpc.CustomCodec(grpctool.RawCodecWithProtoFallback{}), // nolint: staticcheck
	)
	gatewayKasVisitor, err := grpctool.NewStreamVisitor(&GatewayKasResponse{})
	require.NoError(t, err)
	r := &router{
		kasPool: &defaultKasPool{
			dialOpts: []grpc.DialOption{
				grpc.WithInsecure(),
				grpc.WithContextDialer(privateApiServerListener.DialContext),
			},
		},
		tunnelQuerier:     querier,
		tunnelFinder:      finder,
		internalServer:    internalServer,
		privateApiServer:  privateApiServer,
		gatewayKasVisitor: gatewayKasVisitor,
	}
	r.RegisterAgentApi(&test.Testing_ServiceDesc)
	var wg wait.Group
	defer wg.Wait()
	defer internalServer.Stop()
	defer privateApiServer.Stop()
	wg.Start(func() {
		assert.NoError(t, internalServer.Serve(internalServerListener))
	})
	wg.Start(func() {
		assert.NoError(t, privateApiServer.Serve(privateApiServerListener))
	})
	internalServerConn, err := grpc.DialContext(context.Background(), "pipe",
		grpc.WithContextDialer(internalServerListener.DialContext),
		grpc.WithInsecure(),
		grpc.WithChainStreamInterceptor(
			grpctool.StreamClientValidatingInterceptor,
		),
		grpc.WithChainUnaryInterceptor(
			grpctool.UnaryClientValidatingInterceptor,
		),
	)
	require.NoError(t, err)
	client := test.NewTestingClient(internalServerConn)
	runTest(client)
}
