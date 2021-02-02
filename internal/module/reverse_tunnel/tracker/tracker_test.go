package tracker

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/info"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_redis"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	_ Registerer = &RedisTracker{}
	_ Tracker    = &RedisTracker{}
)

func TestRegisterConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, hash, ti := setupTracker(t)

	hash.EXPECT().
		Set(gomock.Any(), ti.AgentId, ti.ConnectionId, gomock.Any()).
		Do(func(ctx context.Context, key interface{}, hashKey int64, value *anypb.Any) {
			cancel()
		})

	go func() {
		assert.True(t, r.RegisterTunnel(context.Background(), ti))
	}()

	require.NoError(t, r.Run(ctx))
}

func TestUnregisterConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, hash, ti := setupTracker(t)

	gomock.InOrder(
		hash.EXPECT().
			Set(gomock.Any(), ti.AgentId, ti.ConnectionId, gomock.Any()),
		hash.EXPECT().
			Unset(gomock.Any(), ti.AgentId, ti.ConnectionId).
			Do(func(ctx context.Context, key interface{}, hashKey int64) {
				cancel()
			}),
	)

	go func() {
		assert.True(t, r.RegisterTunnel(context.Background(), ti))
		assert.True(t, r.UnregisterTunnel(context.Background(), ti))
	}()

	require.NoError(t, r.Run(ctx))
}

func TestGC(t *testing.T) {
	r, hash, _ := setupTracker(t)

	hash.EXPECT().
		GC(gomock.Any()).
		Return(3, nil)

	removed, err := r.runGc(context.Background())
	assert.NoError(t, err)
	assert.EqualValues(t, 3, removed)
}

func TestRefreshRegistrations(t *testing.T) {
	r, hash, _ := setupTracker(t)

	hash.EXPECT().
		Refresh(gomock.Any())
	assert.NoError(t, r.refreshRegistrations(context.Background()))
}

func setupTracker(t *testing.T) (*RedisTracker, *mock_redis.MockExpiringHashInterface, *TunnelInfo) {
	ctrl := gomock.NewController(t)
	hash := mock_redis.NewMockExpiringHashInterface(ctrl)
	ti := &TunnelInfo{
		AgentDescriptor: &info.AgentDescriptor{
			Services: []*info.Service{
				{
					Name: "bla",
					Methods: []*info.Method{
						{
							Name: "bab",
						},
					},
				},
			},
		},
		ConnectionId: 123,
		AgentId:      543,
	}
	return &RedisTracker{
		log:              zaptest.NewLogger(t),
		refreshPeriod:    time.Minute,
		gcPeriod:         time.Minute,
		tunnelsByAgentId: hash,
		toRegister:       make(chan *TunnelInfo),
		toUnregister:     make(chan *TunnelInfo),
	}, hash, ti
}
