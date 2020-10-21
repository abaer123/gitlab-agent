package apiutil

import (
	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type contextKeyType int

const (
	agentMetaKey contextKeyType = iota
)

func AgentMetaFromContext(ctx context.Context) *api.AgentMeta {
	agentMeta, ok := ctx.Value(agentMetaKey).(*api.AgentMeta)
	if !ok {
		// This is a programmer error, so panic.
		panic("*api.AgentMeta not attached to context. Make sure you are using interceptors")
	}
	return agentMeta
}

func AgentMetaFromRawContext(ctx context.Context) (*api.AgentMeta, error) {
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, err
	}
	return &api.AgentMeta{
		Token: api.AgentToken(token),
	}, nil
}

// AgentTokenFromContext extracts the agent token from the given context
func AgentTokenFromContext(ctx context.Context) api.AgentToken {
	return AgentMetaFromContext(ctx).Token
}

func InjectAgentMeta(ctx context.Context, agentMeta *api.AgentMeta) context.Context {
	return context.WithValue(ctx, agentMetaKey, agentMeta)
}

// UnaryAgentMetaInterceptor is a gRPC server-side interceptor that populates context with api.AgentMeta for unary RPCs.
func UnaryAgentMetaInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		agentMeta, err := AgentMetaFromRawContext(ctx)
		if err != nil {
			return nil, err // err is already a status.Error
		}
		return handler(InjectAgentMeta(ctx, agentMeta), req)
	}
}

// StreamAgentMetaInterceptor is a gRPC server-side interceptor that populates context with api.AgentMeta for streaming RPCs.
func StreamAgentMetaInterceptor() func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		agentMeta, err := AgentMetaFromRawContext(ss.Context())
		if err != nil {
			return err // err is already a status.Error
		}
		wrapper := grpc_middleware.WrapServerStream(ss)
		wrapper.WrappedContext = InjectAgentMeta(wrapper.Context(), agentMeta)
		return handler(srv, wrapper)
	}
}

func NewTokenCredentials(token string, insecure bool) credentials.PerRPCCredentials {
	return &tokenCredentials{
		token:    token,
		insecure: insecure,
	}
}

type tokenCredentials struct {
	token    string
	insecure bool
}

func (t *tokenCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		api.MetadataAuthorization: "Bearer " + t.token,
	}, nil
}

func (t *tokenCredentials) RequireTransportSecurity() bool {
	return !t.insecure
}
