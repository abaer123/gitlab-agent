package gitaly_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_internalgitaly"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

const (
	fileMaxSize int64 = 1000
)

var (
	_ gitaly.FileVisitor          = &gitaly.AccumulatingFileVisitor{}
	_ gitaly.FetchVisitor         = gitaly.ChunkingFetchVisitor{}
	_ gitaly.PathFetcherInterface = &gitaly.PathFetcher{}
)

func TestPathFetcherHappyPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	r := repo()
	treeEntriesReq := &gitalypb.GetTreeEntriesRequest{
		Repository: r,
		Revision:   []byte(revision),
		Path:       []byte(repoPath),
		Recursive:  false,
	}
	commitClient := mock_gitaly.NewMockCommitServiceClient(mockCtrl)
	expectedEntry1 := &gitalypb.TreeEntry{
		Path:      []byte("manifest1.yaml"),
		Type:      gitalypb.TreeEntry_BLOB,
		CommitOid: manifestRevision,
	}
	data1 := []byte("data1")
	treeEntry := &gitalypb.TreeEntry{
		Path:      []byte("some_dir"),
		Type:      gitalypb.TreeEntry_TREE,
		CommitOid: manifestRevision,
	}
	expectedEntry2 := &gitalypb.TreeEntry{
		Path:      []byte("manifest2.yaml"),
		Type:      gitalypb.TreeEntry_BLOB,
		CommitOid: manifestRevision,
	}
	data2 := []byte("data2")
	mockGetTreeEntries(t, mockCtrl, commitClient, treeEntriesReq, []*gitalypb.TreeEntry{expectedEntry1, treeEntry, expectedEntry2})
	mockTreeEntry(t, mockCtrl, commitClient, data1, &gitalypb.TreeEntryRequest{
		Repository: r,
		Revision:   []byte(expectedEntry1.CommitOid),
		Path:       expectedEntry1.Path,
		MaxSize:    fileMaxSize,
	})
	mockTreeEntry(t, mockCtrl, commitClient, data2, &gitalypb.TreeEntryRequest{
		Repository: r,
		Revision:   []byte(expectedEntry2.CommitOid),
		Path:       expectedEntry2.Path,
		MaxSize:    fileMaxSize,
	})
	mockVisitor := mock_internalgitaly.NewMockFetchVisitor(mockCtrl)
	gomock.InOrder(
		mockVisitor.EXPECT().
			Entry(matcher.ProtoEq(t, expectedEntry1)).
			Return(true, fileMaxSize, nil),
		mockVisitor.EXPECT().
			StreamChunk(expectedEntry1.Path, data1[:1]).
			Return(false, nil),
		mockVisitor.EXPECT().
			StreamChunk(expectedEntry1.Path, data1[1:]).
			Return(false, nil),
		mockVisitor.EXPECT().
			Entry(matcher.ProtoEq(t, expectedEntry2)).
			Return(true, fileMaxSize, nil),
		mockVisitor.EXPECT().
			StreamChunk(expectedEntry2.Path, data2[:1]).
			Return(false, nil),
		mockVisitor.EXPECT().
			StreamChunk(expectedEntry2.Path, data2[1:]).
			Return(false, nil),
	)
	v := gitaly.PathFetcher{
		Client: commitClient,
	}
	err := v.Visit(context.Background(), r, []byte(revision), []byte(repoPath), false, mockVisitor)
	require.NoError(t, err)
}

func TestChunkingFetchVisitor_Entry(t *testing.T) {
	entry := &gitalypb.TreeEntry{
		Path:      []byte("manifest2.yaml"),
		Type:      gitalypb.TreeEntry_BLOB,
		CommitOid: manifestRevision,
	}
	mockCtrl := gomock.NewController(t)
	fv := mock_internalgitaly.NewMockFetchVisitor(mockCtrl)
	fv.EXPECT().
		Entry(matcher.ProtoEq(t, entry)).
		Return(true, int64(100), nil)
	v := gitaly.ChunkingFetchVisitor{
		MaxChunkSize: 10,
		Delegate:     fv,
	}
	download, maxSize, err := v.Entry(entry)
	assert.True(t, download)
	assert.EqualValues(t, 100, maxSize)
	assert.NoError(t, err)
}

func TestChunkingFetchVisitor_StreamChunk(t *testing.T) {
	t.Run("no chunking", func(t *testing.T) {
		p := []byte{1, 2, 3}
		data := []byte{4, 5, 6}
		mockCtrl := gomock.NewController(t)
		fv := mock_internalgitaly.NewMockFetchVisitor(mockCtrl)
		fv.EXPECT().
			StreamChunk(p, data).
			Return(false, nil)
		v := gitaly.ChunkingFetchVisitor{
			MaxChunkSize: 10,
			Delegate:     fv,
		}
		done, err := v.StreamChunk(p, data)
		assert.False(t, done)
		assert.NoError(t, err)
	})
	t.Run("chunking", func(t *testing.T) {
		p := []byte{1, 2, 3}
		data := []byte{4, 5, 6}
		mockCtrl := gomock.NewController(t)
		fv := mock_internalgitaly.NewMockFetchVisitor(mockCtrl)
		gomock.InOrder(
			fv.EXPECT().
				StreamChunk(p, data[:2]).
				Return(false, nil),
			fv.EXPECT().
				StreamChunk(p, data[2:]).
				Return(false, nil),
		)
		v := gitaly.ChunkingFetchVisitor{
			MaxChunkSize: 2,
			Delegate:     fv,
		}
		done, err := v.StreamChunk(p, data)
		assert.False(t, done)
		assert.NoError(t, err)
	})
	t.Run("done", func(t *testing.T) {
		p := []byte{1, 2, 3}
		data := []byte{4, 5, 6}
		mockCtrl := gomock.NewController(t)
		fv := mock_internalgitaly.NewMockFetchVisitor(mockCtrl)
		fv.EXPECT().
			StreamChunk(p, data[:2]).
			Return(true, nil)
		v := gitaly.ChunkingFetchVisitor{
			MaxChunkSize: 2,
			Delegate:     fv,
		}
		done, err := v.StreamChunk(p, data)
		assert.True(t, done)
		assert.NoError(t, err)
	})
	t.Run("error", func(t *testing.T) {
		p := []byte{1, 2, 3}
		data := []byte{4, 5, 6}
		mockCtrl := gomock.NewController(t)
		fv := mock_internalgitaly.NewMockFetchVisitor(mockCtrl)
		fv.EXPECT().
			StreamChunk(p, data[:2]).
			Return(false, errors.New("boom!"))
		v := gitaly.ChunkingFetchVisitor{
			MaxChunkSize: 2,
			Delegate:     fv,
		}
		done, err := v.StreamChunk(p, data)
		assert.False(t, done)
		assert.EqualError(t, err, "boom!")
	})
}

func mockTreeEntry(t *testing.T, mockCtrl *gomock.Controller, commitClient *mock_gitaly.MockCommitServiceClient, data []byte, req *gitalypb.TreeEntryRequest) {
	treeEntryClient := mock_gitaly.NewMockCommitService_TreeEntryClient(mockCtrl)
	// Emulate streaming response
	resp1 := &gitalypb.TreeEntryResponse{
		Type: gitalypb.TreeEntryResponse_BLOB,
		Data: data[:1],
	}
	resp2 := &gitalypb.TreeEntryResponse{
		Type: gitalypb.TreeEntryResponse_BLOB,
		Data: data[1:],
	}
	gomock.InOrder(
		commitClient.EXPECT().
			TreeEntry(gomock.Any(), matcher.ProtoEq(t, req)).
			Return(treeEntryClient, nil),
		treeEntryClient.EXPECT().
			Recv().
			Return(resp1, nil),
		treeEntryClient.EXPECT().
			Recv().
			Return(resp2, nil),
		treeEntryClient.EXPECT().
			Recv().
			Return(nil, io.EOF),
	)
}
