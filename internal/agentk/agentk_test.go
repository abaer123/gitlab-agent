package agentk

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc/mock_agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap/zaptest"
)

const (
	revision = "rev12341234"
)

func TestGetConfigurationResumeConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a, mockCtrl, client := setupBasicAgent(t)
	configStream1 := mock_agentrpc.NewMockKas_GetConfigurationClient(mockCtrl)
	configStream2 := mock_agentrpc.NewMockKas_GetConfigurationClient(mockCtrl)
	gomock.InOrder(
		client.EXPECT().
			GetConfiguration(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ConfigurationRequest{})).
			Return(configStream1, nil),
		configStream1.EXPECT().
			Recv().
			Return(&agentrpc.ConfigurationResponse{
				Configuration: &agentcfg.AgentConfiguration{},
				CommitId:      revision,
			}, nil),
		configStream1.EXPECT().
			Recv().
			Return(nil, io.EOF),
		client.EXPECT().
			GetConfiguration(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ConfigurationRequest{
				CommitId: revision,
			})).
			Return(configStream2, nil),
		configStream2.EXPECT().
			Recv().
			DoAndReturn(func() (*agentrpc.ConfigurationResponse, error) {
				cancel()
				return nil, io.EOF
			}),
	)
	err := a.Run(ctx)
	require.NoError(t, err)
}

func setupBasicAgent(t *testing.T) (*Agent, *gomock.Controller, *mock_agentrpc.MockKasClient) {
	mockCtrl := gomock.NewController(t)
	client := mock_agentrpc.NewMockKasClient(mockCtrl)
	agent := &Agent{
		Log:                             zaptest.NewLogger(t),
		KasClient:                       client,
		RefreshConfigurationRetryPeriod: 10 * time.Millisecond,
		ModuleFactories:                 nil,
	}
	return agent, mockCtrl, client
}
