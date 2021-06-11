package gitaly_test

import (
	"context"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_internalgitaly"
	"gitlab.com/gitlab-org/gitaly/v14/proto/go/gitalypb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	fileMaxSize int64 = 1000
)

var (
	_ gitaly.FileVisitor          = &gitaly.AccumulatingFileVisitor{}
	_ gitaly.PathFetcherInterface = &gitaly.PathFetcher{}
)

func TestPathFetcher_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	r := repo()
	treeEntriesReq := &gitalypb.GetTreeEntriesRequest{
		Repository: r,
		Revision:   []byte(revision),
		Path:       []byte(repoPath),
		Recursive:  false,
	}
	commitClient := mock_gitaly.NewMockCommitServiceClient(ctrl)
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
	mockGetTreeEntries(t, ctrl, commitClient, treeEntriesReq, []*gitalypb.TreeEntry{expectedEntry1, treeEntry, expectedEntry2})
	mockTreeEntry(t, ctrl, commitClient, data1, &gitalypb.TreeEntryRequest{
		Repository: r,
		Revision:   []byte(expectedEntry1.CommitOid),
		Path:       expectedEntry1.Path,
		MaxSize:    fileMaxSize,
	})
	mockTreeEntry(t, ctrl, commitClient, data2, &gitalypb.TreeEntryRequest{
		Repository: r,
		Revision:   []byte(expectedEntry2.CommitOid),
		Path:       expectedEntry2.Path,
		MaxSize:    fileMaxSize,
	})
	mockVisitor := mock_internalgitaly.NewMockFetchVisitor(ctrl)
	gomock.InOrder(
		mockVisitor.EXPECT().
			Entry(matcher.ProtoEq(t, expectedEntry1)).
			Return(true, fileMaxSize, nil),
		mockVisitor.EXPECT().
			StreamChunk(expectedEntry1.Path, data1[:1]),
		mockVisitor.EXPECT().
			StreamChunk(expectedEntry1.Path, data1[1:]),
		mockVisitor.EXPECT().
			EntryDone(matcher.ProtoEq(t, expectedEntry1), nil),
		mockVisitor.EXPECT().
			Entry(matcher.ProtoEq(t, expectedEntry2)).
			Return(true, fileMaxSize, nil),
		mockVisitor.EXPECT().
			StreamChunk(expectedEntry2.Path, data2[:1]),
		mockVisitor.EXPECT().
			StreamChunk(expectedEntry2.Path, data2[1:]),
		mockVisitor.EXPECT().
			EntryDone(matcher.ProtoEq(t, expectedEntry2), nil),
	)
	v := gitaly.PathFetcher{
		Client: commitClient,
	}
	err := v.Visit(context.Background(), r, []byte(revision), []byte(repoPath), false, mockVisitor)
	require.NoError(t, err)
}

func TestPathFetcher_StreamFile_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	commitClient := mock_gitaly.NewMockCommitServiceClient(ctrl)
	treeEntryClient := mock_gitaly.NewMockCommitService_TreeEntryClient(ctrl)
	mockVisitor := mock_internalgitaly.NewMockFileVisitor(ctrl)
	r := repo()
	req := &gitalypb.TreeEntryRequest{
		Repository: r,
		Revision:   []byte(revision),
		Path:       []byte(repoPath),
		MaxSize:    fileMaxSize,
	}
	gomock.InOrder(
		commitClient.EXPECT().
			TreeEntry(gomock.Any(), matcher.ProtoEq(t, req)).
			Return(treeEntryClient, nil),
		treeEntryClient.EXPECT().
			Recv().
			Return(nil, status.Error(codes.NotFound, "file is not here")),
	)
	v := gitaly.PathFetcher{
		Client: commitClient,
	}
	err := v.StreamFile(context.Background(), r, []byte(revision), []byte(repoPath), fileMaxSize, mockVisitor)
	require.EqualError(t, err, "FileNotFound: TreeEntry.Recv: file/directory/ref not found: dir")
}

func TestPathFetcher_StreamFile_TooBig(t *testing.T) {
	ctrl := gomock.NewController(t)
	commitClient := mock_gitaly.NewMockCommitServiceClient(ctrl)
	treeEntryClient := mock_gitaly.NewMockCommitService_TreeEntryClient(ctrl)
	mockVisitor := mock_internalgitaly.NewMockFileVisitor(ctrl)
	r := repo()
	req := &gitalypb.TreeEntryRequest{
		Repository: r,
		Revision:   []byte(revision),
		Path:       []byte(repoPath),
		MaxSize:    fileMaxSize,
	}
	gomock.InOrder(
		commitClient.EXPECT().
			TreeEntry(gomock.Any(), matcher.ProtoEq(t, req)).
			Return(treeEntryClient, nil),
		treeEntryClient.EXPECT().
			Recv().
			Return(nil, status.Error(codes.FailedPrecondition, "file is too big")),
	)
	v := gitaly.PathFetcher{
		Client: commitClient,
	}
	err := v.StreamFile(context.Background(), r, []byte(revision), []byte(repoPath), fileMaxSize, mockVisitor)
	require.EqualError(t, err, "FileTooBig: TreeEntry: file is too big: dir: rpc error: code = FailedPrecondition desc = file is too big")
}

func TestPathFetcher_StreamFile_InvalidType(t *testing.T) {
	ctrl := gomock.NewController(t)
	commitClient := mock_gitaly.NewMockCommitServiceClient(ctrl)
	treeEntryClient := mock_gitaly.NewMockCommitService_TreeEntryClient(ctrl)
	mockVisitor := mock_internalgitaly.NewMockFileVisitor(ctrl)
	r := repo()
	req := &gitalypb.TreeEntryRequest{
		Repository: r,
		Revision:   []byte(revision),
		Path:       []byte(repoPath),
		MaxSize:    fileMaxSize,
	}
	resp := &gitalypb.TreeEntryResponse{
		Type: gitalypb.TreeEntryResponse_COMMIT,
		Oid:  manifestRevision,
	}
	gomock.InOrder(
		commitClient.EXPECT().
			TreeEntry(gomock.Any(), matcher.ProtoEq(t, req)).
			Return(treeEntryClient, nil),
		treeEntryClient.EXPECT().
			Recv().
			Return(resp, nil),
	)
	v := gitaly.PathFetcher{
		Client: commitClient,
	}
	err := v.StreamFile(context.Background(), r, []byte(revision), []byte(repoPath), fileMaxSize, mockVisitor)
	require.EqualError(t, err, "UnexpectedTreeEntryType: TreeEntry.Recv: file is not a usual file: dir")
}

func mockTreeEntry(t *testing.T, ctrl *gomock.Controller, commitClient *mock_gitaly.MockCommitServiceClient, data []byte, req *gitalypb.TreeEntryRequest) {
	treeEntryClient := mock_gitaly.NewMockCommitService_TreeEntryClient(ctrl)
	// Emulate streaming response
	resp1 := &gitalypb.TreeEntryResponse{
		Type: gitalypb.TreeEntryResponse_BLOB, // only the first response has the type set!
		Data: data[:1],
	}
	resp2 := &gitalypb.TreeEntryResponse{
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
