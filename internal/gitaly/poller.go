package gitaly

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

const (
	DefaultBranch = ""
)

var (
	_ PollerInterface = &Poller{}
)

type PollerInterface interface {
	Poll(ctx context.Context, repo *gitalypb.Repository, lastProcessedCommitId, refName string) (*PollInfo, error)
}

// Poller does the following:
// - polls ref advertisement for updates to the repository
// - detects which is the main branch, if branch or tag name is not specified
// - compares the commit id the branch or tag is referring to with the last processed one
// - returns the information about the change
type Poller struct {
	Client gitalypb.SmartHTTPServiceClient
}

type PollInfo struct {
	UpdateAvailable bool
	CommitId        string
}

// Poll performs a poll on the repository.
// revision can be a branch name or a tag.
// Poll returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
func (p *Poller) Poll(ctx context.Context, repo *gitalypb.Repository, lastProcessedCommitId, refName string) (*PollInfo, error) {
	r, err := p.fetchRefs(ctx, repo)
	if err != nil {
		return nil, err // don't wrap
	}
	refNameTag := "refs/tags/" + refName
	refNameBranch := "refs/heads/" + refName
	var head, master, wanted *Reference

loop:
	for i := range r.Refs {
		switch r.Refs[i].Name {
		case refNameTag, refNameBranch:
			wanted = &r.Refs[i]
			break loop
		case "HEAD":
			head = &r.Refs[i]
		case "refs/heads/master":
			master = &r.Refs[i]
		}
	}
	if wanted == nil { // not found
		if refName != DefaultBranch { // were looking for something specific, but didn't find it
			return nil, fmt.Errorf("ref %q not found", refName)
		}
		// looking for default branch
		if head != nil {
			wanted = head
		} else if master != nil {
			wanted = master
		} else {
			return nil, errors.New("default branch not found")
		}
	}
	return &PollInfo{
		UpdateAvailable: wanted.Oid != lastProcessedCommitId,
		CommitId:        wanted.Oid,
	}, nil
}

// fetchRefs returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
func (p *Poller) fetchRefs(ctx context.Context, repo *gitalypb.Repository) (*ReferenceDiscovery, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // ensure streaming call is canceled
	uploadPackReq := &gitalypb.InfoRefsRequest{
		Repository: repo,
	}
	uploadPackResp, err := p.Client.InfoRefsUploadPack(ctx, uploadPackReq)
	if err != nil {
		return nil, fmt.Errorf("InfoRefsUploadPack: %w", err) // wrap
	}
	var inforefs []byte
	for {
		entry, err := uploadPackResp.Recv() // nolint: govet
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("InfoRefsUploadPack.Recv: %w", err) // wrap
		}
		inforefs = append(inforefs, entry.Data...)
	}
	refs, err := ParseReferenceDiscovery(bytes.NewReader(inforefs))
	if err != nil {
		return nil, err
	}
	return &refs, nil
}
