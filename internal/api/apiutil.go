package api

import (
	"context"
)

type contextKeyType int

const (
	agentMDKey contextKeyType = iota
)

// AgentMDFromContext extracts AgentMD from a context.
func AgentMDFromContext(ctx context.Context) *AgentMD {
	agentMD, ok := ctx.Value(agentMDKey).(*AgentMD)
	if !ok {
		// This is a programmer error, so panic.
		panic("*api.AgentMD not attached to context. Make sure you are using interceptors")
	}
	return agentMD
}

// AgentTokenFromContext extracts the agent token from the given context
func AgentTokenFromContext(ctx context.Context) AgentToken {
	return AgentMDFromContext(ctx).Token
}

// InjectAgentMD injects AgentMD into a context.
func InjectAgentMD(ctx context.Context, agentMD *AgentMD) context.Context {
	return context.WithValue(ctx, agentMDKey, agentMD)
}
