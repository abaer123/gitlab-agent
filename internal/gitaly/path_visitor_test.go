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
	revision         = "507ebc6de9bcac25628aa7afd52802a91a0685d8"
	manifestRevision = "7afd52802a91a0685d8507ebc6de9bcac25628aa"
	repoPath         = "dir"
)

func TestPathVisitorHappyPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	r := repo()
	treeEntriesReq := &gitalypb.GetTreeEntriesRequest{
		Repository: r,
		Revision:   []byte(revision),
		Path:       []byte(repoPath),
		Recursive:  false,
	}
	commitClient := mock_gitaly.NewMockCommitServiceClient(mockCtrl)
	expectedEntry := &gitalypb.TreeEntry{
		Path:      []byte("manifest.yaml"),
		Type:      gitalypb.TreeEntry_BLOB,
		CommitOid: manifestRevision,
	}
	mockGetTreeEntries(t, mockCtrl, commitClient, treeEntriesReq, []*gitalypb.TreeEntry{expectedEntry})
	mockVisitor := mock_internalgitaly.NewMockPathEntryVisitor(mockCtrl)
	mockVisitor.EXPECT().
		Entry(matcher.ProtoEq(t, expectedEntry)).
		Return(false, nil)
	v := gitaly.PathVisitor{
		Client: commitClient,
	}
	err := v.Visit(context.Background(), r, []byte(revision), []byte(repoPath), false, mockVisitor)
	require.NoError(t, err)
}

func mockGetTreeEntries(t *testing.T, mockCtrl *gomock.Controller, commitClient *mock_gitaly.MockCommitServiceClient, req *gitalypb.GetTreeEntriesRequest, entries []*gitalypb.TreeEntry) {
	treeEntriesClient := mock_gitaly.NewMockCommitService_GetTreeEntriesClient(mockCtrl)
	gomock.InOrder(
		commitClient.EXPECT().
			GetTreeEntries(gomock.Any(), matcher.ProtoEq(t, req), gomock.Any()).
			Return(treeEntriesClient, nil),
		treeEntriesClient.EXPECT().
			Recv().
			Return(&gitalypb.GetTreeEntriesResponse{Entries: entries}, nil),
		treeEntriesClient.EXPECT().
			Recv().
			Return(nil, io.EOF),
	)
}

func repo() *gitalypb.Repository {
	return &gitalypb.Repository{
		StorageName:        "StorageName1",
		RelativePath:       "RelativePath1",
		GitObjectDirectory: "GitObjectDirectory1",
		GlRepository:       "GlRepository1",
		GlProjectPath:      "GlProjectPath1",
	}
}
