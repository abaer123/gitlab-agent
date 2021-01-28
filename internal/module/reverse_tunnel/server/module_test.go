package server

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	_ modserver.Module  = &module{}
	_ modserver.Factory = &Factory{}
)

func TestConnectAllowsValidToken(t *testing.T) {
	ctrl, mockApi, h, m := setupModule(t)
	agentInfo := testhelpers.AgentInfoObj()
	ctx := api.InjectAgentMD(context.Background(), &api.AgentMD{Token: testhelpers.AgentkToken})
	ctx = grpctool.InjectLogger(ctx, zaptest.NewLogger(t))
	server := mock_reverse_tunnel.NewMockReverseTunnel_ConnectServer(ctrl)
	gomock.InOrder(
		server.EXPECT().
			Context().
			Return(ctx),
		mockApi.EXPECT().
			GetAgentInfo(gomock.Any(), gomock.Any(), testhelpers.AgentkToken, false).
			Return(agentInfo, nil, false),
		h.EXPECT().
			HandleTunnelConnection(agentInfo, server),
	)
	err := m.Connect(server)
	require.NoError(t, err)
}

func TestConnectRejectsInvalidToken(t *testing.T) {
	ctrl, mockApi, _, m := setupModule(t)
	ctx := api.InjectAgentMD(context.Background(), &api.AgentMD{Token: "invalid"})
	ctx = grpctool.InjectLogger(ctx, zaptest.NewLogger(t))
	server := mock_reverse_tunnel.NewMockReverseTunnel_ConnectServer(ctrl)
	gomock.InOrder(
		server.EXPECT().
			Context().
			Return(ctx),
		mockApi.EXPECT().
			GetAgentInfo(gomock.Any(), gomock.Any(), gomock.Any(), false).
			Return(nil, errors.New("expected err"), true),
	)
	err := m.Connect(server)
	assert.EqualError(t, err, "expected err")
}

func setupModule(t *testing.T) (*gomock.Controller, *mock_modserver.MockAPI, *mock_reverse_tunnel.MockTunnelConnectionHandler, *module) {
	ctrl := gomock.NewController(t)
	h := mock_reverse_tunnel.NewMockTunnelConnectionHandler(ctrl)
	mockApi := mock_modserver.NewMockAPI(ctrl)
	f := Factory{
		TunnelConnectionHandler: h,
	}
	m, err := f.New(&modserver.Config{
		Log:         zaptest.NewLogger(t),
		Api:         mockApi,
		AgentServer: grpc.NewServer(),
	})
	require.NoError(t, err)
	var wg wait.Group
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
	wg.Start(func() {
		assert.NoError(t, m.Run(ctx))
	})
	return ctrl, mockApi, h, m.(*module)
}