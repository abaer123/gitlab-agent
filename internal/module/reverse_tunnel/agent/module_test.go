package agent

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/info"
	"google.golang.org/grpc"
)

var (
	_ modagent.Module     = &module{}
	_ modagent.Factory    = &Factory{}
	_ connectionInterface = &mockConnection{}
)

func TestModule_FeatureDefaultState(t *testing.T) {
	t.Parallel()
	featureChan := make(chan bool)
	mockConn := &mockConnection{}
	m := setupModule(featureChan, mockConn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := m.Run(ctx, nil)
	require.NoError(t, err)
	assert.Zero(t, mockConn.runCalled)
}

func TestModule_FeatureEnabled(t *testing.T) {
	t.Parallel()
	featureChan := make(chan bool, 1)
	featureChan <- true
	mockConn := &mockConnection{}
	m := setupModule(featureChan, mockConn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := m.Run(ctx, nil)
	require.NoError(t, err)
	assert.EqualValues(t, 1, mockConn.runCalled)
}

func TestModule_FeatureEnabledDisabledEnabled(t *testing.T) {
	t.Parallel()
	featureChan := make(chan bool, 3)
	featureChan <- true
	featureChan <- false
	featureChan <- true
	mockConn := &mockConnection{}
	m := setupModule(featureChan, mockConn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := m.Run(ctx, nil)
	require.NoError(t, err)
	assert.EqualValues(t, 2, mockConn.runCalled)
}

func setupModule(featureChan <-chan bool, mockConn *mockConnection) module {
	return module{
		server:         grpc.NewServer(),
		numConnections: 1,
		featureChan:    featureChan,
		connectionFactory: func(descriptor *info.AgentDescriptor) connectionInterface {
			return mockConn
		},
	}
}

type mockConnection struct {
	runCalled int32
}

func (m *mockConnection) Run(ctx context.Context) {
	atomic.AddInt32(&m.runCalled, 1)
	<-ctx.Done()
}
