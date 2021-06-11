package gitaly

import (
	"context"
	"errors"
	"io"

	"gitlab.com/gitlab-org/gitaly/v14/proto/go/gitalypb"
)

type PathEntryVisitor interface {
	Entry(*gitalypb.TreeEntry) (bool /* done? */, error)
}

type PathVisitor struct {
	Client gitalypb.CommitServiceClient
}

func (v *PathVisitor) Visit(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, recursive bool, visitor PathEntryVisitor) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // ensure streaming call is canceled
	entries, err := v.Client.GetTreeEntries(ctx, &gitalypb.GetTreeEntriesRequest{
		Repository: repo,
		Revision:   revision,
		Path:       repoPath,
		Recursive:  recursive,
	})
	if err != nil {
		return NewRpcError(err, "GetTreeEntries", string(repoPath))
	}
entriesLoop:
	for {
		resp, err := entries.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return NewRpcError(err, "GetTreeEntries.Recv", string(repoPath))
		}
		for _, entry := range resp.Entries {
			done, err := visitor.Entry(entry)
			if err != nil {
				return err // don't wrap
			}
			if done {
				break entriesLoop
			}
		}
	}
	return nil
}
