package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_reverse_tunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
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
			HandleTunnel(gomock.Any(), agentInfo, server),
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

func setupModule(t *testing.T) (*gomock.Controller, *mock_modserver.MockAPI, *mock_reverse_tunnel.MockTunnelHandler, *module) {
	ctrl := gomock.NewController(t)
	h := mock_reverse_tunnel.NewMockTunnelHandler(ctrl)
	mockApi := mock_modserver.NewMockAPI(ctrl)
	f := Factory{
		TunnelHandler: h,
	}
	m, err := f.New(&modserver.Config{
		Log: zaptest.NewLogger(t),
		Api: mockApi,
		Config: &kascfg.ConfigurationFile{
			Agent: &kascfg.AgentCF{
				Listen: &kascfg.ListenAgentCF{
					MaxConnectionAge: durationpb.New(time.Minute),
				},
			},
		},
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
