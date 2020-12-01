package server

import (
	"path"
	"strings"

	"github.com/bmatcuk/doublestar/v2"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type objectsToSynchronizeVisitor struct {
	server                 rpc.Gitops_GetObjectsToSynchronizeServer
	glob                   string
	remainingTotalFileSize int64
	fileSizeLimit          int64
	maxNumberOfFiles       uint32
	numberOfFiles          uint32
	sendFailed             bool
}

func (v *objectsToSynchronizeVisitor) Entry(entry *gitalypb.TreeEntry) (bool /* download? */, int64 /* max size */, error) {
	if v.numberOfFiles == v.maxNumberOfFiles {
		return false, 0, errz.NewUserErrorf("maximum number of manifest files limit reached: %d", v.maxNumberOfFiles)
	}
	v.numberOfFiles++
	filename := string(entry.Path)
	if isHiddenDir(filename) {
		return false, 0, nil
	}
	shouldDownload, err := doublestar.Match(v.glob, filename)
	if err != nil {
		return false, 0, errz.NewUserErrorWithCausef(err, "glob %s match failed", v.glob)
	}
	return shouldDownload, minInt64(v.remainingTotalFileSize, v.fileSizeLimit), nil
}

func (v *objectsToSynchronizeVisitor) StreamChunk(path []byte, data []byte) (bool /* done? */, error) {
	v.remainingTotalFileSize -= int64(len(data))
	if v.remainingTotalFileSize < 0 {
		// This should never happen because we told Gitaly the maximum file size that we'd like to get.
		// i.e. we should have gotten an error from Gitaly if file is bigger than the limit.
		return false, status.Error(codes.Internal, "unexpected negative remaining total file size")
	}
	err := v.server.Send(&rpc.ObjectsToSynchronizeResponse{
		Message: &rpc.ObjectsToSynchronizeResponse_Object_{
			Object: &rpc.ObjectsToSynchronizeResponse_Object{
				Source: string(path),
				Data:   data,
			},
		},
	})
	if err != nil {
		v.sendFailed = true
	}
	return false, err
}

// isHiddenDir checks if a file is in a directory, which name starts with a dot.
func isHiddenDir(filename string) bool {
	dir := path.Dir(filename)
	if dir == "." { // root directory special case
		return false
	}
	parts := strings.Split(dir, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
}
