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
)

const (
	revision         = "507ebc6de9bcac25628aa7afd52802a91a0685d8"
	manifestRevision = "7afd52802a91a0685d8507ebc6de9bcac25628aa"
	repoPath         = "dir"
)

func TestPathVisitor_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	r := repo()
	treeEntriesReq := &gitalypb.GetTreeEntriesRequest{
		Repository: r,
		Revision:   []byte(revision),
		Path:       []byte(repoPath),
		Recursive:  false,
	}
	commitClient := mock_gitaly.NewMockCommitServiceClient(ctrl)
	expectedEntry := &gitalypb.TreeEntry{
		Path:      []byte("manifest.yaml"),
		Type:      gitalypb.TreeEntry_BLOB,
		CommitOid: manifestRevision,
	}
	features := map[string]string{
		"f1": "true",
	}
	mockGetTreeEntries(t, ctrl, matcher.GrpcOutgoingCtx(features), commitClient, treeEntriesReq, []*gitalypb.TreeEntry{expectedEntry})
	mockVisitor := mock_internalgitaly.NewMockPathEntryVisitor(ctrl)
	mockVisitor.EXPECT().
		Entry(matcher.ProtoEq(t, expectedEntry))
	v := gitaly.PathVisitor{
		Client:   commitClient,
		Features: features,
	}
	err := v.Visit(context.Background(), r, []byte(revision), []byte(repoPath), false, mockVisitor)
	require.NoError(t, err)
}

func mockGetTreeEntries(t *testing.T, ctrl *gomock.Controller, ctx gomock.Matcher, commitClient *mock_gitaly.MockCommitServiceClient, req *gitalypb.GetTreeEntriesRequest, entries []*gitalypb.TreeEntry) {
	treeEntriesClient := mock_gitaly.NewMockCommitService_GetTreeEntriesClient(ctrl)
	gomock.InOrder(
		commitClient.EXPECT().
			GetTreeEntries(ctx, matcher.ProtoEq(t, req), gomock.Any()).
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
