package gitaly

import (
	"context"
	"errors"
	"io"

	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PathFetcherInterface interface {
	// Visit returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
	// Visit returns *Error when a error occurs.
	Visit(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, recursive bool, visitor FetchVisitor) error
	// StreamFile streams the specified revision of the file.
	// The passed visitor is never called if file was not found.
	// StreamFile returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
	// StreamFile returns *Error when a error occurs.
	StreamFile(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, sizeLimit int64, v FileVisitor) error
	// FetchFile fetches the specified revision of a file.
	// FetchFile returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
	// FetchFile returns *Error when a error occurs.
	FetchFile(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, sizeLimit int64) ([]byte, error)
}

// FileVisitor is the visitor callback, invoked for each chunk of a file.
type FileVisitor interface {
	Chunk(data []byte) (bool /* done? */, error)
}

// FetchVisitor is the visitor callback, invoked for each chunk of each path entry.
type FetchVisitor interface {
	Entry(*gitalypb.TreeEntry) (bool /* download? */, int64 /* max size */, error)
	StreamChunk(path []byte, data []byte) (bool /* done? */, error)
	// EntryDone is called after the entry has been fully streamed.
	// It's not called for entries that are not streamed i.e. skipped.
	EntryDone(*gitalypb.TreeEntry, error)
}

type PathFetcher struct {
	Client gitalypb.CommitServiceClient
}

func (f *PathFetcher) Visit(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, recursive bool, visitor FetchVisitor) error {
	v := PathVisitor{
		Client: f.Client,
	}
	return v.Visit(ctx, repo, revision, repoPath, recursive, fetcherPathEntryVisitor(func(entry *gitalypb.TreeEntry) (bool /* done? */, error) {
		if entry.Type != gitalypb.TreeEntry_BLOB {
			return false, nil
		}
		shouldFetch, maxSize, err := visitor.Entry(entry)
		if err != nil || !shouldFetch {
			return false, err
		}
		err = f.StreamFile(ctx, repo, []byte(entry.CommitOid), entry.Path, maxSize, fetcherFileVisitor(func(data []byte) (bool /* done? */, error) {
			return visitor.StreamChunk(entry.Path, data)
		}))
		visitor.EntryDone(entry, err)
		return false, err
	}))
}

func (f *PathFetcher) StreamFile(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, sizeLimit int64, v FileVisitor) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // ensure streaming call is canceled
	teResp, err := f.Client.TreeEntry(ctx, &gitalypb.TreeEntryRequest{
		Repository: repo,
		Revision:   revision,
		Path:       repoPath,
		MaxSize:    sizeLimit,
	})
	if err != nil {
		return NewRpcError(err, "TreeEntry", string(repoPath))
	}
	firstMessage := true
	for {
		entry, err := teResp.Recv()
		if err != nil {
			code := status.Code(err)
			switch {
			case code == codes.FailedPrecondition:
				return NewFileTooBigError(err, "TreeEntry", string(repoPath))
			case code == codes.NotFound:
				return NewNotFoundError("TreeEntry.Recv", string(repoPath))
			case errors.Is(err, io.EOF):
				return nil
			default:
				return NewRpcError(err, "TreeEntry.Recv", string(repoPath))
			}
		}
		if firstMessage {
			firstMessage = false
			if entry.Type != gitalypb.TreeEntryResponse_BLOB {
				return NewUnexpectedTreeEntryTypeError("TreeEntry.Recv", string(repoPath))
			}
		}
		done, err := v.Chunk(entry.Data)
		if err != nil || done {
			return err
		}
	}
}

func (f *PathFetcher) FetchFile(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, sizeLimit int64) ([]byte, error) {
	v := &AccumulatingFileVisitor{}
	err := f.StreamFile(ctx, repo, revision, repoPath, sizeLimit, v)
	if err != nil {
		return nil, err
	}
	return v.Data, nil
}

type fetcherPathEntryVisitor func(*gitalypb.TreeEntry) (bool /* done? */, error)

func (v fetcherPathEntryVisitor) Entry(entry *gitalypb.TreeEntry) (bool /* done? */, error) {
	return v(entry)
}

type fetcherFileVisitor func(data []byte) (bool /* done? */, error)

func (v fetcherFileVisitor) Chunk(data []byte) (bool /* done? */, error) {
	return v(data)
}

type AccumulatingFileVisitor struct {
	Data []byte
}

func (a *AccumulatingFileVisitor) Chunk(data []byte) (bool /* done? */, error) {
	a.Data = append(a.Data, data...)
	return false, nil
}
