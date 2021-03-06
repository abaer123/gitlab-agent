package rpc_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/testing/protocmp"
)

const (
	revision1 = "rev12341234"
	revision2 = "rev123412341"
)

var (
	_ rpc.ConfigurationWatcherInterface = &rpc.ConfigurationWatcher{}
)

func TestConfigurationWatcher(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockCtrl := gomock.NewController(t)
	client := mock_rpc.NewMockAgentConfigurationClient(mockCtrl)
	configStream := mock_rpc.NewMockAgentConfiguration_GetConfigurationClient(mockCtrl)
	cfg1 := &agentcfg.AgentConfiguration{
		Gitops: &agentcfg.GitopsCF{
			ManifestProjects: []*agentcfg.ManifestProjectCF{
				{
					Id: "bla",
				},
			},
		},
	}
	cfg2 := &agentcfg.AgentConfiguration{}
	gomock.InOrder(
		client.EXPECT().
			GetConfiguration(gomock.Any(), matcher.ProtoEq(t, &rpc.ConfigurationRequest{})).
			Return(configStream, nil),
		configStream.EXPECT().
			Recv().
			Return(&rpc.ConfigurationResponse{
				Configuration: cfg1,
				CommitId:      revision1,
			}, nil),
		configStream.EXPECT().
			Recv().
			Return(&rpc.ConfigurationResponse{
				Configuration: cfg2,
				CommitId:      revision2,
			}, nil),
		configStream.EXPECT().
			Recv().
			DoAndReturn(func() (*rpc.ConfigurationResponse, error) {
				cancel()
				return nil, io.EOF
			}),
	)
	w := rpc.ConfigurationWatcher{
		Log:         zaptest.NewLogger(t),
		Client:      client,
		RetryPeriod: 10 * time.Millisecond,
	}
	iter := 0
	w.Watch(ctx, func(ctx context.Context, config rpc.ConfigurationData) {
		switch iter {
		case 0:
			assert.Empty(t, cmp.Diff(config.Config, cfg1, protocmp.Transform()))
		case 1:
			assert.Empty(t, cmp.Diff(config.Config, cfg2, protocmp.Transform()))
		default:
			t.Fatal(iter)
		}
		iter++
	})
	assert.EqualValues(t, 2, iter)
}

func TestConfigurationWatcherResumeConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockCtrl := gomock.NewController(t)
	client := mock_rpc.NewMockAgentConfigurationClient(mockCtrl)
	configStream1 := mock_rpc.NewMockAgentConfiguration_GetConfigurationClient(mockCtrl)
	configStream2 := mock_rpc.NewMockAgentConfiguration_GetConfigurationClient(mockCtrl)
	gomock.InOrder(
		client.EXPECT().
			GetConfiguration(gomock.Any(), matcher.ProtoEq(t, &rpc.ConfigurationRequest{})).
			Return(configStream1, nil),
		configStream1.EXPECT().
			Recv().
			Return(&rpc.ConfigurationResponse{
				Configuration: &agentcfg.AgentConfiguration{},
				CommitId:      revision1,
			}, nil),
		configStream1.EXPECT().
			Recv().
			Return(nil, io.EOF),
		client.EXPECT().
			GetConfiguration(gomock.Any(), matcher.ProtoEq(t, &rpc.ConfigurationRequest{
				CommitId: revision1,
			})).
			Return(configStream2, nil),
		configStream2.EXPECT().
			Recv().
			DoAndReturn(func() (*rpc.ConfigurationResponse, error) {
				cancel()
				return nil, io.EOF
			}),
	)
	w := rpc.ConfigurationWatcher{
		Log:         zaptest.NewLogger(t),
		Client:      client,
		RetryPeriod: 10 * time.Millisecond,
	}
	w.Watch(ctx, func(ctx context.Context, config rpc.ConfigurationData) {
		// Don't care
	})
}
