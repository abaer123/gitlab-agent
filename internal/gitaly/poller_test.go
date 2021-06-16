package gitaly

import (
	"context"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_gitaly"
	"gitlab.com/gitlab-org/gitaly/v14/proto/go/gitalypb"
)

const (
	revision1 = "507ebc6de9bcac25628aa7afd52802a91a0685d8"
	revision2 = "28aa7afd52802a91a0685d8507ebc6de9bcac256"

	branch = "test-branch"

	infoRefsData = `001e# service=git-upload-pack
00000148` + revision1 + ` HEAD` + "\x00" + `multi_ack thin-pack side-band side-band-64k ofs-delta shallow deepen-since deepen-not deepen-relative no-progress include-tag multi_ack_detailed allow-tip-sha1-in-want allow-reachable-sha1-in-want no-done symref=HEAD:refs/heads/master filter object-format=sha1 agent=git/2.28.0
003f` + revision1 + ` refs/heads/master
0044` + revision2 + ` refs/heads/` + branch + `
0000`
)

func TestPoller(t *testing.T) {
	tests := []struct {
		name                string
		ref                 string
		lastProcessedCommit string
		expectedInfoCommit  string
		expectedInfoUpdate  bool
	}{
		{
			name:                "default branch same commit",
			ref:                 DefaultBranch,
			lastProcessedCommit: revision1,
			expectedInfoCommit:  revision1,
			expectedInfoUpdate:  false,
		},
		{
			name:                "master branch same commit",
			ref:                 "master",
			lastProcessedCommit: revision1,
			expectedInfoCommit:  revision1,
			expectedInfoUpdate:  false,
		},
		{
			name:                "custom branch same commit",
			ref:                 branch,
			lastProcessedCommit: revision2,
			expectedInfoCommit:  revision2,
			expectedInfoUpdate:  false,
		},
		{
			name:                "default branch no commit",
			ref:                 DefaultBranch,
			lastProcessedCommit: "",
			expectedInfoCommit:  revision1,
			expectedInfoUpdate:  true,
		},
		{
			name:                "master branch no commit",
			ref:                 "master",
			lastProcessedCommit: "",
			expectedInfoCommit:  revision1,
			expectedInfoUpdate:  true,
		},
		{
			name:                "custom branch no commit",
			ref:                 branch,
			lastProcessedCommit: "",
			expectedInfoCommit:  revision2,
			expectedInfoUpdate:  true,
		},
		{
			name:                "default branch another commit",
			ref:                 DefaultBranch,
			lastProcessedCommit: "1231232",
			expectedInfoCommit:  revision1,
			expectedInfoUpdate:  true,
		},
		{
			name:                "master branch another commit",
			ref:                 "master",
			lastProcessedCommit: "123123123",
			expectedInfoCommit:  revision1,
			expectedInfoUpdate:  true,
		},
		{
			name:                "custom branch another commit",
			ref:                 branch,
			lastProcessedCommit: "13213123",
			expectedInfoCommit:  revision2,
			expectedInfoUpdate:  true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			r := repo()
			infoRefsReq := &gitalypb.InfoRefsRequest{Repository: r}
			httpClient := mock_gitaly.NewMockSmartHTTPServiceClient(ctrl)
			mockInfoRefsUploadPack(t, ctrl, httpClient, infoRefsReq, []byte(infoRefsData))
			p := Poller{
				Client: httpClient,
			}
			pollInfo, err := p.Poll(context.Background(), r, tc.lastProcessedCommit, tc.ref)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedInfoUpdate, pollInfo.UpdateAvailable)
			assert.Equal(t, tc.expectedInfoCommit, pollInfo.CommitId)
		})
	}
}

func TestPollerErrors(t *testing.T) {
	t.Run("branch not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		r := repo()
		infoRefsReq := &gitalypb.InfoRefsRequest{Repository: r}
		httpClient := mock_gitaly.NewMockSmartHTTPServiceClient(ctrl)
		mockInfoRefsUploadPack(t, ctrl, httpClient, infoRefsReq, []byte(infoRefsData))
		p := Poller{
			Client: httpClient,
		}
		_, err := p.Poll(context.Background(), r, "", "some_branch")
		require.EqualError(t, err, "FileNotFound: InfoRefsUploadPack: file/directory/ref not found: some_branch")
	})
	t.Run("no HEAD", func(t *testing.T) {
		noHEAD := `001e# service=git-upload-pack
00000155` + revision1 + ` refs/heads/master` + "\x00" + `multi_ack thin-pack side-band side-band-64k ofs-delta shallow deepen-since deepen-not deepen-relative no-progress include-tag multi_ack_detailed allow-tip-sha1-in-want allow-reachable-sha1-in-want no-done symref=HEAD:refs/heads/master filter object-format=sha1 agent=git/2.28.0
0044` + revision2 + ` refs/heads/` + branch + `
0000`
		ctrl := gomock.NewController(t)
		r := repo()
		infoRefsReq := &gitalypb.InfoRefsRequest{Repository: r}
		httpClient := mock_gitaly.NewMockSmartHTTPServiceClient(ctrl)
		mockInfoRefsUploadPack(t, ctrl, httpClient, infoRefsReq, []byte(noHEAD))
		p := Poller{
			Client: httpClient,
		}
		pollInfo, err := p.Poll(context.Background(), r, "", DefaultBranch)
		require.NoError(t, err)
		assert.True(t, pollInfo.UpdateAvailable)
		assert.Equal(t, revision1, pollInfo.CommitId)
	})
	t.Run("no HEAD no master", func(t *testing.T) {
		noHEAD := `001e# service=git-upload-pack
00000155` + revision1 + ` refs/heads/bababa` + "\x00" + `multi_ack thin-pack side-band side-band-64k ofs-delta shallow deepen-since deepen-not deepen-relative no-progress include-tag multi_ack_detailed allow-tip-sha1-in-want allow-reachable-sha1-in-want no-done symref=HEAD:refs/heads/master filter object-format=sha1 agent=git/2.28.0
0044` + revision2 + ` refs/heads/` + branch + `
0000`
		ctrl := gomock.NewController(t)
		r := repo()
		infoRefsReq := &gitalypb.InfoRefsRequest{Repository: r}
		httpClient := mock_gitaly.NewMockSmartHTTPServiceClient(ctrl)
		mockInfoRefsUploadPack(t, ctrl, httpClient, infoRefsReq, []byte(noHEAD))
		p := Poller{
			Client: httpClient,
		}
		_, err := p.Poll(context.Background(), r, "", DefaultBranch)
		require.EqualError(t, err, "FileNotFound: InfoRefsUploadPack: file/directory/ref not found: default branch")
	})
}

func mockInfoRefsUploadPack(t *testing.T, ctrl *gomock.Controller, httpClient *mock_gitaly.MockSmartHTTPServiceClient, infoRefsReq *gitalypb.InfoRefsRequest, data []byte) {
	infoRefsClient := mock_gitaly.NewMockSmartHTTPService_InfoRefsUploadPackClient(ctrl)
	// Emulate streaming response
	resp1 := &gitalypb.InfoRefsResponse{
		Data: data[:1],
	}
	resp2 := &gitalypb.InfoRefsResponse{
		Data: data[1:],
	}
	gomock.InOrder(
		infoRefsClient.EXPECT().
			Recv().
			Return(resp1, nil),
		infoRefsClient.EXPECT().
			Recv().
			Return(resp2, nil),
		infoRefsClient.EXPECT().
			Recv().
			Return(nil, io.EOF),
	)
	httpClient.EXPECT().
		InfoRefsUploadPack(gomock.Any(), matcher.ProtoEq(t, infoRefsReq)).
		Return(infoRefsClient, nil)
}

func repo() *gitalypb.Repository {
	return &gitalypb.Repository{
		StorageName:        "StorageName",
		RelativePath:       "RelativePath",
		GitObjectDirectory: "GitObjectDirectory",
		GlRepository:       "GlRepository",
		GlProjectPath:      "GlProjectPath",
	}
}
