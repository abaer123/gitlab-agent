package gitaly_test

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_internalgitaly"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

const (
	defaultGitOpsManifestPathGlob = "**/*.{yaml,yml,json}"
)

var (
	_ gitaly.FetchVisitor = gitaly.ChunkingFetchVisitor{}
	_ gitaly.FetchVisitor = &gitaly.EntryCountLimitingFetchVisitor{}
	_ gitaly.FetchVisitor = &gitaly.TotalSizeLimitingFetchVisitor{}
	_ gitaly.FetchVisitor = gitaly.HiddenDirFilteringFetchVisitor{}
	_ gitaly.FetchVisitor = gitaly.GlobFilteringFetchVisitor{}
	_ gitaly.FetchVisitor = gitaly.DuplicatePathDetectingVisitor{}
	_ error               = &gitaly.GlobMatchFailedError{}
	_ error               = &gitaly.MaxNumberOfFilesError{}
	_ error               = &gitaly.DuplicatePathFoundError{}
)

func TestChunkingFetchVisitor_Entry(t *testing.T) {
	entry, fv := delegate(t)
	v := gitaly.NewChunkingFetchVisitor(fv, 10)
	download, maxSize, err := v.Entry(entry)
	assert.True(t, download)
	assert.EqualValues(t, 100, maxSize)
	assert.NoError(t, err)
}

func TestChunkingFetchVisitor_StreamChunk(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		p := []byte{}
		data := []byte{}
		ctrl := gomock.NewController(t)
		fv := mock_internalgitaly.NewMockFetchVisitor(ctrl)
		fv.EXPECT().
			StreamChunk(p, data)
		v := gitaly.NewChunkingFetchVisitor(fv, 10)
		done, err := v.StreamChunk(p, data)
		assert.False(t, done)
		assert.NoError(t, err)
	})
	t.Run("no chunking", func(t *testing.T) {
		p := []byte{1, 2, 3}
		data := []byte{4, 5, 6}
		ctrl := gomock.NewController(t)
		fv := mock_internalgitaly.NewMockFetchVisitor(ctrl)
		fv.EXPECT().
			StreamChunk(p, data)
		v := gitaly.NewChunkingFetchVisitor(fv, 10)
		done, err := v.StreamChunk(p, data)
		assert.False(t, done)
		assert.NoError(t, err)
	})
	t.Run("chunking", func(t *testing.T) {
		p := []byte{1, 2, 3}
		data := []byte{4, 5, 6}
		ctrl := gomock.NewController(t)
		fv := mock_internalgitaly.NewMockFetchVisitor(ctrl)
		gomock.InOrder(
			fv.EXPECT().
				StreamChunk(p, data[:2]),
			fv.EXPECT().
				StreamChunk(p, data[2:]),
		)
		v := gitaly.NewChunkingFetchVisitor(fv, 2)
		done, err := v.StreamChunk(p, data)
		assert.False(t, done)
		assert.NoError(t, err)
	})
	t.Run("done", func(t *testing.T) {
		p := []byte{1, 2, 3}
		data := []byte{4, 5, 6}
		ctrl := gomock.NewController(t)
		fv := mock_internalgitaly.NewMockFetchVisitor(ctrl)
		fv.EXPECT().
			StreamChunk(p, data[:2]).
			Return(true, nil)
		v := gitaly.NewChunkingFetchVisitor(fv, 2)
		done, err := v.StreamChunk(p, data)
		assert.True(t, done)
		assert.NoError(t, err)
	})
	t.Run("error", func(t *testing.T) {
		p := []byte{1, 2, 3}
		data := []byte{4, 5, 6}
		ctrl := gomock.NewController(t)
		fv := mock_internalgitaly.NewMockFetchVisitor(ctrl)
		fv.EXPECT().
			StreamChunk(p, data[:2]).
			Return(false, errors.New("boom!"))
		v := gitaly.NewChunkingFetchVisitor(fv, 2)
		done, err := v.StreamChunk(p, data)
		assert.False(t, done)
		assert.EqualError(t, err, "boom!")
	})
}

func TestEntryCountLimitingFetchVisitor(t *testing.T) {
	p := []byte{1, 2, 3}
	data := []byte{4, 5, 6}
	entry, fv := delegate(t)
	fv.EXPECT().
		StreamChunk(p, data)
	fv.EXPECT().
		EntryDone(matcher.ProtoEq(t, entry), nil)

	v := gitaly.NewEntryCountLimitingFetchVisitor(fv, 1)
	download, maxSize, err := v.Entry(entry)
	require.NoError(t, err)
	assert.EqualValues(t, 100, maxSize)
	assert.True(t, download)
	assert.EqualValues(t, 1, v.FilesVisited)
	assert.Zero(t, v.FilesSent)

	done, err := v.StreamChunk(p, data)
	require.NoError(t, err)
	assert.False(t, done)
	assert.Zero(t, v.FilesSent)

	v.EntryDone(entry, nil)
	assert.EqualValues(t, 1, v.FilesSent)

	_, _, err = v.Entry(&gitalypb.TreeEntry{})
	assert.EqualError(t, err, "maximum number of files limit reached: 1")
	assert.EqualValues(t, 1, v.FilesVisited) // still 1
	assert.EqualValues(t, 1, v.FilesSent)    // still 1
}

func TestTotalSizeLimitingFetchVisitor(t *testing.T) {
	p := []byte{1, 2, 3}
	data := []byte{4, 5, 6}
	entry, fv := delegate(t)
	fv.EXPECT().
		StreamChunk(p, data)

	v := gitaly.NewTotalSizeLimitingFetchVisitor(fv, 50)
	download, maxSize, err := v.Entry(entry)
	require.NoError(t, err)
	assert.EqualValues(t, 50, maxSize)
	assert.True(t, download)
	assert.EqualValues(t, 50, v.RemainingTotalFileSize) // still 50

	done, err := v.StreamChunk(p, data)
	require.NoError(t, err)
	assert.False(t, done)
	assert.EqualValues(t, 47, v.RemainingTotalFileSize)
}

func TestTotalSizeLimitingFetchVisitor_Underflow(t *testing.T) {
	p := []byte{1, 2, 3}
	data := []byte{4, 5, 6}
	entry, fv := delegate(t)

	v := gitaly.NewTotalSizeLimitingFetchVisitor(fv, 2)
	_, _, err := v.Entry(entry)
	require.NoError(t, err)

	_, err = v.StreamChunk(p, data)
	assert.EqualError(t, err, "rpc error: code = Internal desc = unexpected negative remaining total file size")
}

func TestHiddenDirFilteringFetchVisitor(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		expectedDownload bool
	}{
		{
			path:             ".dir/manifest1.yaml",
			expectedDownload: false,
		},
		{
			path:             "dir1/.dir2/manifest1.yaml",
			expectedDownload: false,
		},
		{
			path:             "manifest1.yaml",
			expectedDownload: true,
		},
		{
			path:             "dir1/manifest1.yaml",
			expectedDownload: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			entry := &gitalypb.TreeEntry{
				Path: []byte(tc.path), // nolint: scopelint
			}
			ctrl := gomock.NewController(t)
			fv := mock_internalgitaly.NewMockFetchVisitor(ctrl)
			if tc.expectedDownload { // nolint: scopelint
				fv.EXPECT().
					Entry(matcher.ProtoEq(t, entry)).
					Return(true, int64(100), nil)
			}
			v := gitaly.NewHiddenDirFilteringFetchVisitor(fv)
			download, _, err := v.Entry(entry)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedDownload, download) // nolint: scopelint
		})
	}
}

func TestGlobFilteringFetchVisitor(t *testing.T) {
	tests := []struct {
		path             string
		glob             string
		expectedDownload bool
	}{
		{
			path:             "manifest1.yaml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
		},
		{
			path:             "manifest1.yml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
		},
		{
			path:             "manifest1.json",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
		},
		{
			path:             "dir/manifest1.yaml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
		},
		{
			path:             "dir/manifest1.yml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
		},
		{
			path:             "dir/manifest1.json",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
		},
		{
			path:             "manifest1.txt",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: false,
		},
		{
			path:             "dir/manifest1.txt",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: false,
		},
		{
			path:             "dir1/manifest1.yaml",
			glob:             "**.yaml", // yes, this does not match "dir/file" names. See https://github.com/bmatcuk/doublestar/issues/48
			expectedDownload: false,
		},
		{
			path:             "dir1/manifest1.yaml",
			glob:             "dir2/*.yml",
			expectedDownload: false,
		},
		{
			path:             "manifest1.yaml",
			glob:             "**.yaml",
			expectedDownload: true,
		},
		{
			path:             "manifest1.yaml",
			glob:             "**",
			expectedDownload: true,
		},
		{
			path:             "dir1/manifest1.yaml",
			glob:             "**",
			expectedDownload: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			entry := &gitalypb.TreeEntry{
				Path: []byte(tc.path), // nolint: scopelint
			}
			ctrl := gomock.NewController(t)
			fv := mock_internalgitaly.NewMockFetchVisitor(ctrl)
			if tc.expectedDownload { // nolint: scopelint
				fv.EXPECT().
					Entry(matcher.ProtoEq(t, entry)).
					Return(true, int64(100), nil)
			}
			v := gitaly.NewGlobFilteringFetchVisitor(fv, tc.glob) // nolint: scopelint
			download, _, err := v.Entry(entry)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedDownload, download) // nolint: scopelint
		})
	}
}

func TestDuplicatePathDetectingVisitor(t *testing.T) {
	entry, fv := delegate(t)

	v := gitaly.NewDuplicateFileDetectingVisitor(fv)
	download, maxSize, err := v.Entry(entry)
	require.NoError(t, err)
	assert.EqualValues(t, 100, maxSize)
	assert.True(t, download)

	_, _, err = v.Entry(entry)
	require.EqualError(t, err, "path visited more than once: manifest.yaml")
}

func delegate(t *testing.T) (*gitalypb.TreeEntry, *mock_internalgitaly.MockFetchVisitor) {
	entry := &gitalypb.TreeEntry{
		Path:      []byte("manifest.yaml"),
		Type:      gitalypb.TreeEntry_BLOB,
		CommitOid: manifestRevision,
	}
	ctrl := gomock.NewController(t)
	fv := mock_internalgitaly.NewMockFetchVisitor(ctrl)
	fv.EXPECT().
		Entry(matcher.ProtoEq(t, entry)).
		Return(true, int64(100), nil)
	return entry, fv
}
