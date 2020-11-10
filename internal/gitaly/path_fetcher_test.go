package gitaly_test

import (
	"context"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly/mock_internalgitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/mock_gitaly"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

const (
	fileMaxSize int64 = 1000
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
		Limit:      fileMaxSize,
	})
	mockTreeEntry(t, mockCtrl, commitClient, data2, &gitalypb.TreeEntryRequest{
		Repository: r,
		Revision:   []byte(expectedEntry2.CommitOid),
		Path:       expectedEntry2.Path,
		Limit:      fileMaxSize,
	})
	mockVisitor := mock_internalgitaly.NewMockFetchVisitor(mockCtrl)
	gomock.InOrder(
		mockVisitor.EXPECT().
			VisitEntry(matcher.ProtoEq(t, expectedEntry1)).
			Return(true, fileMaxSize, nil),
		mockVisitor.EXPECT().
			VisitBlob(gitaly.Blob{
				Path: expectedEntry1.Path,
				Data: data1,
			}).
			Return(false, nil),
		mockVisitor.EXPECT().
			VisitEntry(matcher.ProtoEq(t, expectedEntry2)).
			Return(true, fileMaxSize, nil),
		mockVisitor.EXPECT().
			VisitBlob(gitaly.Blob{
				Path: expectedEntry2.Path,
				Data: data2,
			}).
			Return(false, nil),
	)
	v := gitaly.PathFetcher{
		Client: commitClient,
	}
	err := v.Visit(context.Background(), r, []byte(revision), []byte(repoPath), false, mockVisitor)
	require.NoError(t, err)
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
