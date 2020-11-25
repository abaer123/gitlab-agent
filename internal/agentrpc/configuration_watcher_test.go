package agentrpc_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc/mock_agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/testing/protocmp"
)

const (
	revision1 = "rev12341234"
	revision2 = "rev123412341"
)

var (
	_ agentrpc.ConfigurationWatcherInterface = &agentrpc.ConfigurationWatcher{}
)

func TestConfigurationWatcher(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockCtrl := gomock.NewController(t)
	client := mock_agentrpc.NewMockKasClient(mockCtrl)
	configStream := mock_agentrpc.NewMockKas_GetConfigurationClient(mockCtrl)
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
			GetConfiguration(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ConfigurationRequest{})).
			Return(configStream, nil),
		configStream.EXPECT().
			Recv().
			Return(&agentrpc.ConfigurationResponse{
				Configuration: cfg1,
				CommitId:      revision1,
			}, nil),
		configStream.EXPECT().
			Recv().
			Return(&agentrpc.ConfigurationResponse{
				Configuration: cfg2,
				CommitId:      revision2,
			}, nil),
		configStream.EXPECT().
			Recv().
			DoAndReturn(func() (*agentrpc.ConfigurationResponse, error) {
				cancel()
				return nil, io.EOF
			}),
	)
	w := agentrpc.ConfigurationWatcher{
		Log:         zaptest.NewLogger(t),
		KasClient:   client,
		RetryPeriod: 10 * time.Millisecond,
	}
	iter := 0
	w.Watch(ctx, func(ctx context.Context, commitId string, configuration *agentcfg.AgentConfiguration) {
		switch iter {
		case 0:
			assert.Empty(t, cmp.Diff(configuration, cfg1, protocmp.Transform()))
		case 1:
			assert.Empty(t, cmp.Diff(configuration, cfg2, protocmp.Transform()))
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
	client := mock_agentrpc.NewMockKasClient(mockCtrl)
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
				CommitId:      revision1,
			}, nil),
		configStream1.EXPECT().
			Recv().
			Return(nil, io.EOF),
		client.EXPECT().
			GetConfiguration(gomock.Any(), matcher.ProtoEq(t, &agentrpc.ConfigurationRequest{
				CommitId: revision1,
			})).
			Return(configStream2, nil),
		configStream2.EXPECT().
			Recv().
			DoAndReturn(func() (*agentrpc.ConfigurationResponse, error) {
				cancel()
				return nil, io.EOF
			}),
	)
	w := agentrpc.ConfigurationWatcher{
		Log:         zaptest.NewLogger(t),
		KasClient:   client,
		RetryPeriod: 10 * time.Millisecond,
	}
	w.Watch(ctx, func(ctx context.Context, commitId string, configuration *agentcfg.AgentConfiguration) {
		// Don't care
	})
}
