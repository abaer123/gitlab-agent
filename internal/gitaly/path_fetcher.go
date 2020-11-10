package gitaly

import (
	"context"
	"errors"
	"fmt"
	"io"

	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

// FetchVisitor is the visitor callback, invoked for each path entry.
type FetchVisitor interface {
	VisitEntry(*gitalypb.TreeEntry) (bool /* download? */, int64 /* max size */, error)
	VisitBlob(Blob) (bool /* done? */, error)
}

type Blob struct {
	Path []byte
	Data []byte
}

type PathFetcher struct {
	Client gitalypb.CommitServiceClient
}

func (f *PathFetcher) Visit(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, recursive bool, visitor FetchVisitor) error {
	v := PathVisitor{
		Client: f.Client,
	}
	return v.Visit(ctx, repo, revision, repoPath, recursive, fetcherPathVisitor(func(entry *gitalypb.TreeEntry) (bool /* done? */, error) {
		if entry.Type != gitalypb.TreeEntry_BLOB {
			return false, nil
		}
		shouldFetch, maxSize, err := visitor.VisitEntry(entry)
		if err != nil {
			return false, err
		}
		if !shouldFetch {
			return false, nil
		}
		file, err := f.FetchSingleFile(ctx, repo, []byte(entry.CommitOid), entry.Path, maxSize)
		if err != nil {
			return false, err // don't wrap
		}
		return visitor.VisitBlob(Blob{
			Path: entry.Path,
			Data: file,
		})
	}))
}

// FetchSingleFile fetches the specified revision of the file.
// Returned data slice is nil if file was not found and is empty if the file is empty.
// FetchSingleFile returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
func (f *PathFetcher) FetchSingleFile(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, sizeLimit int64) ([]byte, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // ensure streaming call is canceled
	teResp, err := f.Client.TreeEntry(ctx, &gitalypb.TreeEntryRequest{
		Repository: repo,
		Revision:   revision,
		Path:       repoPath,
		Limit:      sizeLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("TreeEntry: %w", err) // wrap
	}
	var fileData []byte
	for {
		entry, err := teResp.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("TreeEntry.Recv: %w", err) // wrap
		}
		if entry.Type != gitalypb.TreeEntryResponse_BLOB {
			return nil, fmt.Errorf("TreeEntry: expected BLOB got %s", entry.Type.String())
		}
		fileData = append(fileData, entry.Data...)
	}
	return fileData, nil
}

type fetcherPathVisitor func(*gitalypb.TreeEntry) (bool /* done? */, error)

func (v fetcherPathVisitor) VisitEntry(entry *gitalypb.TreeEntry) (bool /* done? */, error) {
	return v(entry)
}
