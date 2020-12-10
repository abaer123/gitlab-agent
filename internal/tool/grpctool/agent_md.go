package grpctool

import (
	"context"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"google.golang.org/grpc"
)

func AgentMDFromRawContext(ctx context.Context) (*api.AgentMD, error) {
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, err
	}
	return &api.AgentMD{
		Token: api.AgentToken(token),
	}, nil
}

// UnaryServerAgentMDInterceptor is a gRPC server-side interceptor that populates context with api.AgentMD for unary RPCs.
func UnaryServerAgentMDInterceptor() grpc.UnaryServerInterceptor {
	return UnaryServerCtxAugmentingInterceptor(func(ctx context.Context) (context.Context, error) {
		agentMD, err := AgentMDFromRawContext(ctx)
		if err != nil {
			return nil, err // err is already a status.Error
		}
		return api.InjectAgentMD(ctx, agentMD), nil
	})
}

// StreamServerAgentMDInterceptor is a gRPC server-side interceptor that populates context with api.AgentMD for streaming RPCs.
func StreamServerAgentMDInterceptor() grpc.StreamServerInterceptor {
	return StreamServerCtxAugmentingInterceptor(func(ctx context.Context) (context.Context, error) {
		agentMD, err := AgentMDFromRawContext(ctx)
		if err != nil {
			return nil, err // err is already a status.Error
		}
		return api.InjectAgentMD(ctx, agentMD), nil
	})
}
