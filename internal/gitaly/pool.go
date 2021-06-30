package gitaly

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/gitaly/v14/proto/go/gitalypb"
	"google.golang.org/grpc"
)

var (
	_ PoolInterface = &Pool{}
)

type PoolInterface interface {
	Poller(context.Context, *api.GitalyInfo) (PollerInterface, error)
	PathFetcher(context.Context, *api.GitalyInfo) (PathFetcherInterface, error)
}

// ClientPool abstracts gitlab.com/gitlab-org/gitaly/client.Pool.
type ClientPool interface {
	Dial(ctx context.Context, address, token string) (*grpc.ClientConn, error)
}

type Pool struct {
	ClientPool ClientPool
}

func (p *Pool) commitServiceClient(ctx context.Context, info *api.GitalyInfo) (gitalypb.CommitServiceClient, error) {
	conn, err := p.ClientPool.Dial(ctx, info.Address, info.Token)
	if err != nil {
		return nil, err // don't wrap
	}
	return gitalypb.NewCommitServiceClient(conn), nil
}

func (p *Pool) smartHTTPServiceClient(ctx context.Context, info *api.GitalyInfo) (gitalypb.SmartHTTPServiceClient, error) {
	conn, err := p.ClientPool.Dial(ctx, info.Address, info.Token)
	if err != nil {
		return nil, err // don't wrap
	}
	return gitalypb.NewSmartHTTPServiceClient(conn), nil
}

func (p *Pool) PathFetcher(ctx context.Context, info *api.GitalyInfo) (PathFetcherInterface, error) {
	client, err := p.commitServiceClient(ctx, info)
	if err != nil {
		return nil, err
	}
	return &PathFetcher{
		Client:   client,
		Features: info.Features,
	}, nil
}

func (p *Pool) Poller(ctx context.Context, info *api.GitalyInfo) (PollerInterface, error) {
	client, err := p.smartHTTPServiceClient(ctx, info)
	if err != nil {
		return nil, err
	}
	return &Poller{
		Client:   client,
		Features: info.Features,
	}, nil
}
