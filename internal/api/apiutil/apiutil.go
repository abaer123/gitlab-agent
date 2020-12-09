package apiutil

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
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

// AgentTokenFromContext extracts the agent token from the given context
func AgentTokenFromContext(ctx context.Context) api.AgentToken {
	return AgentMetaFromContext(ctx).Token
}

func InjectAgentMeta(ctx context.Context, agentMeta *api.AgentMeta) context.Context {
	return context.WithValue(ctx, agentMetaKey, agentMeta)
}
