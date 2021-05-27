package server

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type objectsToSynchronizeVisitor struct {
	server        rpc.Gitops_GetObjectsToSynchronizeServer
	fileSizeLimit int64
	sendFailed    bool
}

func (v *objectsToSynchronizeVisitor) Entry(entry *gitalypb.TreeEntry) (bool /* download? */, int64 /* max size */, error) {
	return true, v.fileSizeLimit, nil
}

func (v *objectsToSynchronizeVisitor) StreamChunk(path []byte, data []byte) (bool /* done? */, error) {
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

func (v *objectsToSynchronizeVisitor) EntryDone(entry *gitalypb.TreeEntry, err error) {
}
