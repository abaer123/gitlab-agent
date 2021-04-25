package gitaly

import (
	"fmt"
	"path"
	"strings"

	"github.com/bmatcuk/doublestar/v2"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type delegatingFetchVisitor struct {
	delegate FetchVisitor
}

func (v delegatingFetchVisitor) Entry(entry *gitalypb.TreeEntry) (bool /* download? */, int64 /* max size */, error) {
	return v.delegate.Entry(entry)
}

func (v delegatingFetchVisitor) StreamChunk(path []byte, data []byte) (bool /* done? */, error) {
	return v.delegate.StreamChunk(path, data)
}

func (v delegatingFetchVisitor) EntryDone(entry *gitalypb.TreeEntry, err error) {
	v.delegate.EntryDone(entry, err)
}

type ChunkingFetchVisitor struct {
	delegatingFetchVisitor
	maxChunkSize int
}

func NewChunkingFetchVisitor(delegate FetchVisitor, maxChunkSize int) *ChunkingFetchVisitor {
	return &ChunkingFetchVisitor{
		delegatingFetchVisitor: delegatingFetchVisitor{
			delegate: delegate,
		},
		maxChunkSize: maxChunkSize,
	}
}

func (v ChunkingFetchVisitor) StreamChunk(path []byte, data []byte) (bool /* done? */, error) {
	for {
		bytesToSend := minInt(len(data), v.maxChunkSize)
		done, err := v.delegate.StreamChunk(path, data[:bytesToSend])
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

type MaxNumberOfFilesError struct {
	MaxNumberOfFiles uint32
}

func (e *MaxNumberOfFilesError) Error() string {
	return fmt.Sprintf("maximum number of files limit reached: %d", e.MaxNumberOfFiles)
}

type EntryCountLimitingFetchVisitor struct {
	delegatingFetchVisitor
	maxNumberOfFiles uint32
	FilesVisited     uint32
	FilesSent        uint32
}

func NewEntryCountLimitingFetchVisitor(delegate FetchVisitor, maxNumberOfFiles uint32) *EntryCountLimitingFetchVisitor {
	return &EntryCountLimitingFetchVisitor{
		delegatingFetchVisitor: delegatingFetchVisitor{
			delegate: delegate,
		},
		maxNumberOfFiles: maxNumberOfFiles,
	}
}

func (v *EntryCountLimitingFetchVisitor) Entry(entry *gitalypb.TreeEntry) (bool /* download? */, int64 /* max size */, error) {
	if v.FilesVisited == v.maxNumberOfFiles {
		return false, 0, &MaxNumberOfFilesError{
			MaxNumberOfFiles: v.maxNumberOfFiles,
		}
	}
	v.FilesVisited++
	return v.delegate.Entry(entry)
}

func (v *EntryCountLimitingFetchVisitor) EntryDone(entry *gitalypb.TreeEntry, err error) {
	v.delegate.EntryDone(entry, err)
	if err != nil {
		return
	}
	v.FilesSent++
}

type TotalSizeLimitingFetchVisitor struct {
	delegatingFetchVisitor
	RemainingTotalFileSize int64
}

func NewTotalSizeLimitingFetchVisitor(delegate FetchVisitor, maxTotalFileSize int64) *TotalSizeLimitingFetchVisitor {
	return &TotalSizeLimitingFetchVisitor{
		delegatingFetchVisitor: delegatingFetchVisitor{
			delegate: delegate,
		},
		RemainingTotalFileSize: maxTotalFileSize,
	}
}

func (v *TotalSizeLimitingFetchVisitor) Entry(entry *gitalypb.TreeEntry) (bool /* download? */, int64 /* max size */, error) {
	shouldDownload, maxSize, err := v.delegate.Entry(entry)
	if err != nil || !shouldDownload {
		return false, 0, err
	}
	return true, minInt64(v.RemainingTotalFileSize, maxSize), nil
}

func (v *TotalSizeLimitingFetchVisitor) StreamChunk(path []byte, data []byte) (bool /* done? */, error) {
	v.RemainingTotalFileSize -= int64(len(data))
	if v.RemainingTotalFileSize < 0 {
		// This should never happen because we told Gitaly the maximum file size that we'd like to get.
		// i.e. we should have gotten an error from Gitaly if file is bigger than the limit.
		return false, status.Error(codes.Internal, "unexpected negative remaining total file size")
	}
	return v.delegate.StreamChunk(path, data)
}

type HiddenDirFilteringFetchVisitor struct {
	delegatingFetchVisitor
}

func NewHiddenDirFilteringFetchVisitor(delegate FetchVisitor) *HiddenDirFilteringFetchVisitor {
	return &HiddenDirFilteringFetchVisitor{
		delegatingFetchVisitor: delegatingFetchVisitor{
			delegate: delegate,
		},
	}
}

func (v HiddenDirFilteringFetchVisitor) Entry(entry *gitalypb.TreeEntry) (bool /* download? */, int64 /* max size */, error) {
	if isHiddenDir(string(entry.Path)) {
		return false, 0, nil
	}
	return v.delegate.Entry(entry)
}

type GlobMatchFailedError struct {
	Cause error
	Glob  string
}

func (e *GlobMatchFailedError) Error() string {
	return fmt.Sprintf("glob %s match failed: %v", e.Glob, e.Cause)
}

func (e *GlobMatchFailedError) Unwrap() error {
	return e.Cause
}

type GlobFilteringFetchVisitor struct {
	delegatingFetchVisitor
	Glob string
}

func NewGlobFilteringFetchVisitor(delegate FetchVisitor, glob string) *GlobFilteringFetchVisitor {
	return &GlobFilteringFetchVisitor{
		delegatingFetchVisitor: delegatingFetchVisitor{
			delegate: delegate,
		},
		Glob: glob,
	}
}

func (v GlobFilteringFetchVisitor) Entry(entry *gitalypb.TreeEntry) (bool /* download? */, int64 /* max size */, error) {
	shouldDownload, err := doublestar.Match(v.Glob, string(entry.Path))
	if err != nil {
		return false, 0, &GlobMatchFailedError{
			Cause: err,
			Glob:  v.Glob,
		}
	}
	if !shouldDownload {
		return false, 0, nil
	}
	return v.delegate.Entry(entry)
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

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}
