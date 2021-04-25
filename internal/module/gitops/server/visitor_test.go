package server

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

func TestObjectsToSynchronizeVisitor(t *testing.T) {
	p := "asdasd"
	data := []byte{1, 2, 3}
	ctrl := gomock.NewController(t)
	server := mock_rpc.NewMockGitops_GetObjectsToSynchronizeServer(ctrl)
	server.EXPECT().
		Send(matcher.ProtoEq(t, &rpc.ObjectsToSynchronizeResponse{
			Message: &rpc.ObjectsToSynchronizeResponse_Object_{
				Object: &rpc.ObjectsToSynchronizeResponse_Object{
					Source: p,
					Data:   data,
				},
			},
		}))

	v := objectsToSynchronizeVisitor{
		server:        server,
		fileSizeLimit: 100,
	}
	download, maxSize, err := v.Entry(&gitalypb.TreeEntry{
		Path: []byte(p),
	})
	require.NoError(t, err)
	assert.EqualValues(t, 100, maxSize)
	assert.True(t, download)

	done, err := v.StreamChunk([]byte(p), data)
	require.NoError(t, err)
	assert.False(t, done)
	assert.False(t, v.sendFailed)
}

func TestObjectsToSynchronizeVisitor_Error(t *testing.T) {
	p := "asdasd"
	data := []byte{1, 2, 3}
	ctrl := gomock.NewController(t)
	server := mock_rpc.NewMockGitops_GetObjectsToSynchronizeServer(ctrl)
	server.EXPECT().
		Send(gomock.Any()).
		Return(errors.New("expected error"))

	v := objectsToSynchronizeVisitor{
		server:        server,
		fileSizeLimit: 100,
	}
	_, err := v.StreamChunk([]byte(p), data)
	require.EqualError(t, err, "expected error")
	assert.True(t, v.sendFailed)
}
