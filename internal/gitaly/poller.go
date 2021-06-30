package gitaly

import (
	"bytes"
	"context"
	"errors"
	"io"

	"gitlab.com/gitlab-org/gitaly/v14/proto/go/gitalypb"
)

const (
	DefaultBranch = ""
)

var (
	_ PollerInterface = &Poller{}
)

// PollerInterface does the following:
// - polls ref advertisement for updates to the repository
// - detects which is the main branch, if branch or tag name is not specified
// - compares the commit id the branch or tag is referring to with the last processed one
// - returns the information about the change
type PollerInterface interface {
	// Poll performs a poll on the repository.
	// revision can be a branch name or a tag.
	// Poll returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
	// Poll returns *Error when a error occurs.
	Poll(ctx context.Context, repo *gitalypb.Repository, lastProcessedCommitId, refName string) (*PollInfo, error)
}

type Poller struct {
	Client   gitalypb.SmartHTTPServiceClient
	Features map[string]string
}

type PollInfo struct {
	UpdateAvailable bool
	CommitId        string
}

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
			return nil, NewNotFoundError("InfoRefsUploadPack", refName)
		}
		// looking for default branch
		if head != nil {
			wanted = head
		} else if master != nil {
			wanted = master
		} else {
			return nil, NewNotFoundError("InfoRefsUploadPack", "default branch")
		}
	}
	return &PollInfo{
		UpdateAvailable: wanted.Oid != lastProcessedCommitId,
		CommitId:        wanted.Oid,
	}, nil
}

// fetchRefs returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
// fetchRefs returns *Error when a error occurs.
func (p *Poller) fetchRefs(ctx context.Context, repo *gitalypb.Repository) (*ReferenceDiscovery, error) {
	ctx, cancel := context.WithCancel(appendFeatureFlagsToContext(ctx, p.Features))
	defer cancel() // ensure streaming call is canceled
	uploadPackReq := &gitalypb.InfoRefsRequest{
		Repository: repo,
	}
	uploadPackResp, err := p.Client.InfoRefsUploadPack(ctx, uploadPackReq)
	if err != nil {
		return nil, NewRpcError(err, "InfoRefsUploadPack", "")
	}
	var inforefs []byte
	for {
		entry, err := uploadPackResp.Recv() // nolint: govet
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, NewRpcError(err, "InfoRefsUploadPack.Recv", "")
		}
		inforefs = append(inforefs, entry.Data...)
	}
	refs, err := ParseReferenceDiscovery(bytes.NewReader(inforefs))
	if err != nil {
		return nil, NewProtocolError(err, "failed to parse reference discovery", "", "")
	}
	return &refs, nil
}
