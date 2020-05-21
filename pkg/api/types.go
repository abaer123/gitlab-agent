package api

import (
	"context"

	"google.golang.org/grpc"
)

// AgentInfo describes information encoded in the agent token.
type AgentInfo struct {
	// Id is agent's identity.
	// Can contain only /a-z\d-/
	Id        string
	ClusterId int32
	ProjectId int32
	Token     string
}

func AgentInfoFromStream(stream grpc.ServerStream) (*AgentInfo, error) {
	return AgentInfoFromContext(stream.Context())
}

func AgentInfoFromContext(ctx context.Context) (*AgentInfo, error) {
	//md, ok := metadata.FromIncomingContext(ctx)
	// TODO decode token
	return &AgentInfo{
		Id: "agent-ash2k",
	}, nil
}
