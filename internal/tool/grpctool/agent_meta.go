package grpctool

import (
	"context"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api/apiutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	metaAgentVersion      = "agent_version"
	metaAgentCommitId     = "agent_commit_id"
	metaAgentPodNamespace = "agent_pod_namespace"
	metaAgentPodName      = "agent_pod_name"
)

func AgentMetaFromRawContext(ctx context.Context) (*api.AgentMeta, error) {
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, err
	}
	md := metautils.ExtractIncoming(ctx)
	return &api.AgentMeta{
		Token: api.AgentToken(token),
		ClientAgentMeta: api.ClientAgentMeta{
			Version:      md.Get(metaAgentVersion),
			CommitId:     md.Get(metaAgentCommitId),
			PodNamespace: md.Get(metaAgentPodNamespace),
			PodName:      md.Get(metaAgentPodName),
		},
	}, nil
}

// UnaryServerAgentMetaInterceptor is a gRPC server-side interceptor that populates context with api.AgentMeta for unary RPCs.
func UnaryServerAgentMetaInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return UnaryServerCtxAugmentingInterceptor(func(ctx context.Context) (context.Context, error) {
		agentMeta, err := AgentMetaFromRawContext(ctx)
		if err != nil {
			return nil, err // err is already a status.Error
		}
		return apiutil.InjectAgentMeta(ctx, agentMeta), nil
	})
}

// StreamServerAgentMetaInterceptor is a gRPC server-side interceptor that populates context with api.AgentMeta for streaming RPCs.
func StreamServerAgentMetaInterceptor() func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return StreamServerCtxAugmentingInterceptor(func(ctx context.Context) (context.Context, error) {
		agentMeta, err := AgentMetaFromRawContext(ctx)
		if err != nil {
			return nil, err // err is already a status.Error
		}
		return apiutil.InjectAgentMeta(ctx, agentMeta), nil
	})
}

// UnaryClientAgentMetaInterceptor returns a new unary client interceptor that augments connection context with client agent metadata.
func UnaryClientAgentMetaInterceptor(agentMeta *api.ClientAgentMeta) grpc.UnaryClientInterceptor {
	return UnaryClientCtxAugmentingInterceptor(func(ctx context.Context) (context.Context, error) {
		return appendClientAgentMeta(ctx, agentMeta), nil
	})
}

// StreamClientLimitingInterceptor returns a new stream client interceptor that augments connection context with client agent metadata.
func StreamClientAgentMetaInterceptor(agentMeta *api.ClientAgentMeta) grpc.StreamClientInterceptor {
	return StreamClientCtxAugmentingInterceptor(func(ctx context.Context) (context.Context, error) {
		return appendClientAgentMeta(ctx, agentMeta), nil
	})
}

func appendClientAgentMeta(ctx context.Context, agentMeta *api.ClientAgentMeta) context.Context {
	return metadata.AppendToOutgoingContext(ctx,
		metaAgentVersion, agentMeta.Version,
		metaAgentCommitId, agentMeta.CommitId,
		metaAgentPodNamespace, agentMeta.PodNamespace,
		metaAgentPodName, agentMeta.PodName,
	)
}
