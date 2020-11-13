package gitaly

import (
	"context"
	"errors"
	"fmt"
	"io"

	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type PathFetcherInterface interface {
	Visit(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, recursive bool, visitor FetchVisitor) error
	StreamFile(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, sizeLimit int64, v FileVisitor) error
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
		return false, err
	}))
}

// StreamFile streams the specified revision of the file.
// The passed visitor is never called if file was not found.
// StreamFile returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
func (f *PathFetcher) StreamFile(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, sizeLimit int64, v FileVisitor) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // ensure streaming call is canceled
	teResp, err := f.Client.TreeEntry(ctx, &gitalypb.TreeEntryRequest{
		Repository: repo,
		Revision:   revision,
		Path:       repoPath,
		Limit:      sizeLimit,
	})
	if err != nil {
		return fmt.Errorf("TreeEntry: %w", err) // wrap
	}
	for {
		entry, err := teResp.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("TreeEntry.Recv: %w", err) // wrap
		}
		if entry.Type != gitalypb.TreeEntryResponse_BLOB {
			return fmt.Errorf("TreeEntry: expected BLOB got %s", entry.Type.String())
		}
		done, err := v.Chunk(entry.Data)
		if err != nil || done {
			return err
		}
	}
	return nil
}

// FetchFile fetches the specified revision of a file.
// Returned data slice is nil if file was not found and is empty if the file is empty.
// FetchFile returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
func (f *PathFetcher) FetchFile(ctx context.Context, repo *gitalypb.Repository, revision, repoPath []byte, sizeLimit int64) ([]byte, error) {
	v := &AccumulatingFileVisitor{}
	err := f.StreamFile(ctx, repo, revision, repoPath, sizeLimit, v)
	if err != nil {
		return nil, err
	}
	return v.Data, nil
}

type ChunkingFetchVisitor struct {
	MaxChunkSize int
	Delegate     FetchVisitor
}

func (v ChunkingFetchVisitor) Entry(entry *gitalypb.TreeEntry) (bool /* download? */, int64 /* max size */, error) {
	return v.Delegate.Entry(entry)
}

func (v ChunkingFetchVisitor) StreamChunk(path []byte, data []byte) (bool /* done? */, error) {
	for {
		bytesToSend := minInt(len(data), v.MaxChunkSize)
		done, err := v.Delegate.StreamChunk(path, data[:bytesToSend])
		if err != nil || done {
			return done, err
		}
		data = data[bytesToSend:]
		if len(data) == 0 {
			break
		}
	}
	return false, nil
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

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}
