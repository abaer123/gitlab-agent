package server

import (
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
	tests := []struct {
		name             string
		path             string
		glob             string
		expectedDownload bool
		expectedMaxSize  int64
		expectedErr      string
	}{
		{
			name:             "YAML file",
			path:             "manifest1.yaml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "YML file",
			path:             "manifest1.yml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "JSON file",
			path:             "manifest1.json",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "nested YAML file",
			path:             "dir/manifest1.yaml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "nested YML file",
			path:             "dir/manifest1.yml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "nested JSON file",
			path:             "dir/manifest1.json",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "TXT file",
			path:             "manifest1.txt",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: false,
		},
		{
			name:             "nested TXT file",
			path:             "dir/manifest1.txt",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: false,
		},
		{
			name:             "hidden directory",
			path:             ".dir/manifest1.yaml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: false,
		},
		{
			name:             "hidden nested directory",
			path:             "dir1/.dir2/manifest1.yaml",
			glob:             defaultGitOpsManifestPathGlob,
			expectedDownload: false,
		},
		{
			name:             "invalid glob",
			path:             "dir1/manifest1.yaml",
			glob:             "**.yaml", // yes, this does not match "dir/file" names. See https://github.com/bmatcuk/doublestar/issues/48
			expectedDownload: false,
		},
		{
			name:             "no match",
			path:             "dir1/manifest1.yaml",
			glob:             "dir2/*.yml",
			expectedDownload: false,
		},
		{
			name:             "weird glob",
			path:             "manifest1.yaml",
			glob:             "**.yaml",
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "all files 1",
			path:             "manifest1.yaml",
			glob:             "**",
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
		{
			name:             "all files 2",
			path:             "dir1/manifest1.yaml",
			glob:             "**",
			expectedDownload: true,
			expectedMaxSize:  defaultGitopsMaxManifestFileSize,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := objectsToSynchronizeVisitor{
				glob:                   tc.glob, // nolint: scopelint
				remainingTotalFileSize: defaultGitopsMaxTotalManifestFileSize,
				fileSizeLimit:          defaultGitopsMaxManifestFileSize,
				maxNumberOfFiles:       defaultGitopsMaxNumberOfFiles,
			}
			download, maxSize, err := v.Entry(&gitalypb.TreeEntry{
				Path: []byte(tc.path), // nolint: scopelint
			})
			if tc.expectedErr == "" { // nolint: scopelint
				assert.Equal(t, tc.expectedDownload, download) // nolint: scopelint
				if tc.expectedDownload {                       // nolint: scopelint
					assert.Equal(t, tc.expectedMaxSize, maxSize) // nolint: scopelint
				}
			} else {
				assert.EqualError(t, err, tc.expectedErr) // nolint: scopelint
			}
		})
	}
	t.Run("too many files", func(t *testing.T) {
		v := objectsToSynchronizeVisitor{
			glob:                   defaultGitOpsManifestPathGlob,
			remainingTotalFileSize: defaultGitopsMaxTotalManifestFileSize,
			fileSizeLimit:          defaultGitopsMaxManifestFileSize,
			maxNumberOfFiles:       1,
		}
		download, maxSize, err := v.Entry(&gitalypb.TreeEntry{
			Path: []byte("manifest1.yaml"),
		})
		require.NoError(t, err)
		assert.EqualValues(t, defaultGitopsMaxManifestFileSize, maxSize)
		assert.True(t, download)

		_, _, err = v.Entry(&gitalypb.TreeEntry{
			Path: []byte("manifest2.yaml"),
		})
		assert.EqualError(t, err, "maximum number of manifest files limit reached: 1")
	})
	t.Run("unexpected underflow", func(t *testing.T) {
		v := objectsToSynchronizeVisitor{
			glob:                   defaultGitOpsManifestPathGlob,
			remainingTotalFileSize: 1,
			fileSizeLimit:          defaultGitopsMaxManifestFileSize,
			maxNumberOfFiles:       defaultGitopsMaxNumberOfFiles,
		}
		_, err := v.StreamChunk([]byte("manifest2.yaml"), []byte("data1"))
		assert.EqualError(t, err, "rpc error: code = Internal desc = unexpected negative remaining total file size")
	})
	t.Run("blob", func(t *testing.T) {
		data := []byte("data1")
		ctrl := gomock.NewController(t)
		server := mock_rpc.NewMockGitops_GetObjectsToSynchronizeServer(ctrl)
		server.EXPECT().
			Send(matcher.ProtoEq(t, &rpc.ObjectsToSynchronizeResponse{
				Message: &rpc.ObjectsToSynchronizeResponse_Object_{
					Object: &rpc.ObjectsToSynchronizeResponse_Object{
						Source: "manifest2.yaml",
						Data:   data,
					},
				},
			}))
		v := objectsToSynchronizeVisitor{
			server:                 server,
			glob:                   defaultGitOpsManifestPathGlob,
			remainingTotalFileSize: defaultGitopsMaxTotalManifestFileSize,
			fileSizeLimit:          defaultGitopsMaxManifestFileSize,
			maxNumberOfFiles:       defaultGitopsMaxNumberOfFiles,
		}
		done, err := v.StreamChunk([]byte("manifest2.yaml"), data)
		require.NoError(t, err)
		assert.False(t, done)
		assert.EqualValues(t, defaultGitopsMaxTotalManifestFileSize-len(data), v.remainingTotalFileSize)
	})
}
