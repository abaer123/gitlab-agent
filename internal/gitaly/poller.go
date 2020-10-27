package gitaly

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

const (
	DefaultBranch = ""
)

// Poller does the following:
// - polls ref advertisement for updates to the repository
// - detects which is the main branch, if branch or tag name is not specified
// - compares the commit id the branch or tag is referring to with the last processed one
// - returns the information about the change
type Poller struct {
	GitalyPool PoolInterface
}

type PollInfo struct {
	UpdateAvailable bool
	CommitId        string
}

// Poll performs a poll on the repository.
// revision can be a branch name or a tag.
func (p *Poller) Poll(ctx context.Context, gInfo *api.GitalyInfo, repo *gitalypb.Repository, lastProcessedCommitId, refName string) (*PollInfo, error) {
	r, err := p.fetchRefs(ctx, gInfo, repo)
	if err != nil {
		return nil, err
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

func (p *Poller) fetchRefs(ctx context.Context, gInfo *api.GitalyInfo, repo *gitalypb.Repository) (*ReferenceDiscovery, error) {
	client, err := p.GitalyPool.SmartHTTPServiceClient(ctx, gInfo)
	if err != nil {
		return nil, fmt.Errorf("SmartHTTPServiceClient: %v", err)
	}
	uploadPackReq := &gitalypb.InfoRefsRequest{
		Repository: repo,
	}
	uploadPackResp, err := client.InfoRefsUploadPack(ctx, uploadPackReq)
	if err != nil {
		return nil, fmt.Errorf("InfoRefsUploadPack: %v", err)
	}
	var inforefs []byte
	for {
		entry, err := uploadPackResp.Recv() // nolint: govet
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("InfoRefsUploadPack.Recv: %v", err)
		}
		inforefs = append(inforefs, entry.Data...)
	}
	refs, err := ParseReferenceDiscovery(bytes.NewReader(inforefs))
	if err != nil {
		return nil, err
	}
	return &refs, nil
}
