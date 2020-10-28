package gitaly

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc"
)

type PoolInterface interface {
	CommitServiceClient(context.Context, *api.GitalyInfo) (gitalypb.CommitServiceClient, error)
	SmartHTTPServiceClient(context.Context, *api.GitalyInfo) (gitalypb.SmartHTTPServiceClient, error)
}

// ClientPool abstracts gitlab.com/gitlab-org/gitaly/client.Pool.
type ClientPool interface {
	Dial(ctx context.Context, address, token string) (*grpc.ClientConn, error)
}

type Pool struct {
	ClientPool ClientPool
}

func (p *Pool) CommitServiceClient(ctx context.Context, gInfo *api.GitalyInfo) (gitalypb.CommitServiceClient, error) {
	conn, err := p.ClientPool.Dial(ctx, gInfo.Address, gInfo.Token)
	if err != nil {
		return nil, err // don't wrap
	}
	return gitalypb.NewCommitServiceClient(conn), nil
}

func (p *Pool) SmartHTTPServiceClient(ctx context.Context, gInfo *api.GitalyInfo) (gitalypb.SmartHTTPServiceClient, error) {
	conn, err := p.ClientPool.Dial(ctx, gInfo.Address, gInfo.Token)
	if err != nil {
		return nil, err // don't wrap
	}
	return gitalypb.NewSmartHTTPServiceClient(conn), nil
}
