package agent_tracker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modshared"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/redistool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_redis"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	_ Registerer                 = &RedisTracker{}
	_ Querier                    = &RedisTracker{}
	_ Tracker                    = &RedisTracker{}
	_ ConnectedAgentInfoCallback = (&ConnectedAgentInfoCollector{}).Collect
)

func TestRegisterConnection_HappyPath(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, byAgentId, byProjectId, info := setupTracker(t)

	byProjectId.EXPECT().
		Set(gomock.Any(), info.ProjectId, info.ConnectionId, gomock.Any())
	byAgentId.EXPECT().
		Set(gomock.Any(), info.AgentId, info.ConnectionId, gomock.Any()).
		Do(func(ctx context.Context, key interface{}, hashKey int64, value *anypb.Any) {
			cancel()
		})

	go func() {
		assert.True(t, r.RegisterConnection(context.Background(), info))
	}()

	require.NoError(t, r.Run(ctx))
}

func TestRegisterConnection_BothCalledOnError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, byAgentId, byProjectId, info := setupTracker(t)

	byProjectId.EXPECT().
		Set(gomock.Any(), info.ProjectId, info.ConnectionId, gomock.Any()).
		Return(errors.New("err1"))
	byAgentId.EXPECT().
		Set(gomock.Any(), info.AgentId, info.ConnectionId, gomock.Any()).
		DoAndReturn(func(ctx context.Context, key interface{}, hashKey int64, value *anypb.Any) error {
			cancel()
			return errors.New("err2")
		})

	go func() {
		assert.True(t, r.RegisterConnection(context.Background(), info))
	}()

	require.NoError(t, r.Run(ctx))
}

func TestUnregisterConnection_HappyPath(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, byAgentId, byProjectId, info := setupTracker(t)

	gomock.InOrder(
		byProjectId.EXPECT().
			Set(gomock.Any(), info.ProjectId, info.ConnectionId, gomock.Any()),
		byProjectId.EXPECT().
			Unset(gomock.Any(), info.ProjectId, info.ConnectionId),
	)
	gomock.InOrder(
		byAgentId.EXPECT().
			Set(gomock.Any(), info.AgentId, info.ConnectionId, gomock.Any()),
		byAgentId.EXPECT().
			Unset(gomock.Any(), info.AgentId, info.ConnectionId).
			Do(func(ctx context.Context, key interface{}, hashKey int64) {
				cancel()
			}),
	)

	go func() {
		assert.True(t, r.RegisterConnection(context.Background(), info))
		assert.True(t, r.UnregisterConnection(context.Background(), info))
	}()

	require.NoError(t, r.Run(ctx))
}

func TestUnregisterConnection_BothCalledOnError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, byAgentId, byProjectId, info := setupTracker(t)

	gomock.InOrder(
		byProjectId.EXPECT().
			Set(gomock.Any(), info.ProjectId, info.ConnectionId, gomock.Any()),
		byProjectId.EXPECT().
			Unset(gomock.Any(), info.ProjectId, info.ConnectionId).
			Return(errors.New("err1")),
	)
	gomock.InOrder(
		byAgentId.EXPECT().
			Set(gomock.Any(), info.AgentId, info.ConnectionId, gomock.Any()),
		byAgentId.EXPECT().
			Unset(gomock.Any(), info.AgentId, info.ConnectionId).
			DoAndReturn(func(ctx context.Context, key interface{}, hashKey int64) error {
				cancel()
				return errors.New("err1")
			}),
	)

	go func() {
		assert.True(t, r.RegisterConnection(context.Background(), info))
		assert.True(t, r.UnregisterConnection(context.Background(), info))
	}()

	require.NoError(t, r.Run(ctx))
}

func TestGC_HappyPath(t *testing.T) {
	r, byAgentId, byProjectId, _ := setupTracker(t)

	byAgentId.EXPECT().
		GC(gomock.Any()).
		Return(2, nil)

	byProjectId.EXPECT().
		GC(gomock.Any()).
		Return(1, nil)

	removed, err := r.runGc(context.Background())
	assert.NoError(t, err)
	assert.EqualValues(t, 3, removed)
}

func TestGC_BothCalledOnError(t *testing.T) {
	r, byAgentId, byProjectId, _ := setupTracker(t)

	byAgentId.EXPECT().
		GC(gomock.Any()).
		Return(2, errors.New("err1"))

	byProjectId.EXPECT().
		GC(gomock.Any()).
		Return(1, errors.New("err2"))

	removed, err := r.runGc(context.Background())
	assert.Error(t, err)
	assert.EqualValues(t, 3, removed)
}

func TestRefresh_HappyPath(t *testing.T) {
	r, byAgentId, byProjectId, _ := setupTracker(t)

	byAgentId.EXPECT().
		Refresh(gomock.Any())
	byProjectId.EXPECT().
		Refresh(gomock.Any())
	assert.NoError(t, r.refreshRegistrations(context.Background()))
}

func TestRefresh_BothCalledOnError(t *testing.T) {
	r, byAgentId, byProjectId, _ := setupTracker(t)

	byAgentId.EXPECT().
		Refresh(gomock.Any()).
		Return(errors.New("err1"))
	byProjectId.EXPECT().
		Refresh(gomock.Any()).
		Return(errors.New("err2"))
	assert.Error(t, r.refreshRegistrations(context.Background()))
}

func TestGetConnectionsByProjectId_HappyPath(t *testing.T) {
	r, _, byProjectId, info := setupTracker(t)
	any, err := anypb.New(info)
	require.NoError(t, err)
	byProjectId.EXPECT().
		Scan(gomock.Any(), info.ProjectId, gomock.Any()).
		Do(func(ctx context.Context, key interface{}, cb redistool.ScanCallback) (int, error) {
			var done bool
			done, err = cb(any, nil)
			if err != nil || done {
				return 0, err
			}
			return 0, nil
		})
	var cbCalled int
	err = r.GetConnectionsByProjectId(context.Background(), info.ProjectId, func(i *ConnectedAgentInfo) (done bool, err error) {
		cbCalled++
		assert.Empty(t, cmp.Diff(i, info, protocmp.Transform()))
		return false, nil
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, cbCalled)
}

func TestGetConnectionsByProjectId_ScanError(t *testing.T) {
	r, _, byProjectId, info := setupTracker(t)
	byProjectId.EXPECT().
		Scan(gomock.Any(), info.ProjectId, gomock.Any()).
		Do(func(ctx context.Context, key interface{}, cb redistool.ScanCallback) (int, error) {
			done, err := cb(nil, errors.New("intended error"))
			require.NoError(t, err)
			assert.False(t, done)
			return 0, nil
		})
	err := r.GetConnectionsByProjectId(context.Background(), info.ProjectId, func(i *ConnectedAgentInfo) (done bool, err error) {
		require.FailNow(t, "unexpected call")
		return false, nil
	})
	require.NoError(t, err)
}

func TestGetConnectionsByProjectId_UnmarshalError(t *testing.T) {
	r, _, byProjectId, info := setupTracker(t)
	byProjectId.EXPECT().
		Scan(gomock.Any(), info.ProjectId, gomock.Any()).
		Do(func(ctx context.Context, key interface{}, cb redistool.ScanCallback) (int, error) {
			done, err := cb(&anypb.Any{
				TypeUrl: "gitlab.agent.agent_tracker.ConnectedAgentInfo", // valid
				Value:   []byte{1, 2, 3},                                 // invalid
			}, nil)
			require.NoError(t, err) // ignores error to keep going
			assert.False(t, done)
			return 0, nil
		})
	err := r.GetConnectionsByProjectId(context.Background(), info.ProjectId, func(i *ConnectedAgentInfo) (done bool, err error) {
		require.FailNow(t, "unexpected call")
		return false, nil
	})
	require.NoError(t, err)
}

func TestGetConnectionsByAgentId_HappyPath(t *testing.T) {
	r, byAgentId, _, info := setupTracker(t)
	any, err := anypb.New(info)
	require.NoError(t, err)
	byAgentId.EXPECT().
		Scan(gomock.Any(), info.AgentId, gomock.Any()).
		Do(func(ctx context.Context, key interface{}, cb redistool.ScanCallback) (int, error) {
			var done bool
			done, err = cb(any, nil)
			if err != nil || done {
				return 0, err
			}
			return 0, nil
		})
	var cbCalled int
	err = r.GetConnectionsByAgentId(context.Background(), info.AgentId, func(i *ConnectedAgentInfo) (done bool, err error) {
		cbCalled++
		assert.Empty(t, cmp.Diff(i, info, protocmp.Transform()))
		return false, nil
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, cbCalled)
}

func TestGetConnectionsByAgentId_ScanError(t *testing.T) {
	r, byAgentId, _, info := setupTracker(t)
	byAgentId.EXPECT().
		Scan(gomock.Any(), info.AgentId, gomock.Any()).
		Do(func(ctx context.Context, key interface{}, cb redistool.ScanCallback) (int, error) {
			done, err := cb(nil, errors.New("intended error"))
			require.NoError(t, err)
			assert.False(t, done)
			return 0, nil
		})
	err := r.GetConnectionsByAgentId(context.Background(), info.AgentId, func(i *ConnectedAgentInfo) (done bool, err error) {
		require.FailNow(t, "unexpected call")
		return false, nil
	})
	require.NoError(t, err)
}

func TestGetConnectionsByAgentId_UnmarshalError(t *testing.T) {
	r, byAgentId, _, info := setupTracker(t)
	byAgentId.EXPECT().
		Scan(gomock.Any(), info.AgentId, gomock.Any()).
		Do(func(ctx context.Context, key interface{}, cb redistool.ScanCallback) (int, error) {
			done, err := cb(&anypb.Any{
				TypeUrl: "gitlab.agent.agent_tracker.ConnectedAgentInfo", // valid
				Value:   []byte{1, 2, 3},                                 // invalid
			}, nil)
			require.NoError(t, err) // ignores error to keep going
			assert.False(t, done)
			return 0, nil
		})
	err := r.GetConnectionsByAgentId(context.Background(), info.AgentId, func(i *ConnectedAgentInfo) (done bool, err error) {
		require.FailNow(t, "unexpected call")
		return false, nil
	})
	require.NoError(t, err)
}

func setupTracker(t *testing.T) (*RedisTracker, *mock_redis.MockExpiringHashInterface, *mock_redis.MockExpiringHashInterface, *ConnectedAgentInfo) {
	ctrl := gomock.NewController(t)
	byAgentId := mock_redis.NewMockExpiringHashInterface(ctrl)
	byProjectId := mock_redis.NewMockExpiringHashInterface(ctrl)
	tr := &RedisTracker{
		log:                    zaptest.NewLogger(t),
		refreshPeriod:          time.Minute,
		gcPeriod:               time.Minute,
		connectionsByAgentId:   byAgentId,
		connectionsByProjectId: byProjectId,
		toRegister:             make(chan *ConnectedAgentInfo),
		toUnregister:           make(chan *ConnectedAgentInfo),
	}
	return tr, byAgentId, byProjectId, connInfo()
}

func connInfo() *ConnectedAgentInfo {
	return &ConnectedAgentInfo{
		AgentMeta: &modshared.AgentMeta{
			Version:      "v1.2.3",
			CommitId:     "123123",
			PodNamespace: "ns",
			PodName:      "name",
		},
		ConnectedAt:  timestamppb.Now(),
		ConnectionId: 123,
		AgentId:      345,
		ProjectId:    456,
	}
}
