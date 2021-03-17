package tracker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/info"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/redistool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_redis"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	_ Registerer                  = &RedisTracker{}
	_ Tracker                     = &RedisTracker{}
	_ Querier                     = &RedisTracker{}
	_ GetTunnelsByAgentIdCallback = (&TunnelInfoCollector{}).Collect
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

func TestGetTunnelsByAgentId_HappyPath(t *testing.T) {
	r, hash, ti := setupTracker(t)
	any, err := anypb.New(ti)
	require.NoError(t, err)
	hash.EXPECT().
		Scan(gomock.Any(), ti.AgentId, gomock.Any()).
		Do(func(ctx context.Context, key interface{}, cb redistool.ScanCallback) (int, error) {
			var done bool
			done, err = cb(any, nil)
			if err != nil || done {
				return 0, err
			}
			return 0, nil
		})
	var cbCalled int
	err = r.GetTunnelsByAgentId(context.Background(), ti.AgentId, func(tunnelInfo *TunnelInfo) (bool, error) {
		cbCalled++
		assert.Empty(t, cmp.Diff(tunnelInfo, ti, protocmp.Transform()))
		return false, nil
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, cbCalled)
}

func TestGetTunnelsByAgentId_ScanError(t *testing.T) {
	r, hash, ti := setupTracker(t)
	hash.EXPECT().
		Scan(gomock.Any(), ti.AgentId, gomock.Any()).
		Do(func(ctx context.Context, key interface{}, cb redistool.ScanCallback) (int, error) {
			done, err := cb(nil, errors.New("intended error"))
			require.NoError(t, err)
			assert.False(t, done)
			return 0, nil
		})
	err := r.GetTunnelsByAgentId(context.Background(), ti.AgentId, func(tunnelInfo *TunnelInfo) (bool, error) {
		require.FailNow(t, "unexpected call")
		return false, nil
	})
	require.NoError(t, err)
}

func TestGetTunnelsByAgentId_UnmarshalError(t *testing.T) {
	r, hash, ti := setupTracker(t)
	hash.EXPECT().
		Scan(gomock.Any(), ti.AgentId, gomock.Any()).
		Do(func(ctx context.Context, key interface{}, cb redistool.ScanCallback) (int, error) {
			done, err := cb(&anypb.Any{
				TypeUrl: "gitlab.agent.reverse_tunnel.tracker.TunnelInfo", // valid
				Value:   []byte{1, 2, 3},                                  // invalid
			}, nil)
			require.NoError(t, err) // ignores error to keep going
			assert.False(t, done)
			return 0, nil
		})
	err := r.GetTunnelsByAgentId(context.Background(), ti.AgentId, func(tunnelInfo *TunnelInfo) (bool, error) {
		require.FailNow(t, "unexpected call")
		return false, nil
	})
	require.NoError(t, err)
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

func TestTunnelInfoSize(t *testing.T) {
	infoAny, err := anypb.New(&TunnelInfo{
		AgentDescriptor: &info.AgentDescriptor{
			Services: []*info.Service{},
		},
		ConnectionId: 1231232,
		AgentId:      123123,
		KasUrl:       "grpcs://123.123.123.123:123",
	})
	require.NoError(t, err)
	data, err := proto.Marshal(&redistool.ExpiringValue{
		ExpiresAt: timestamppb.Now(),
		Value:     infoAny,
	})
	require.NoError(t, err)
	t.Log(len(data))
}
