package api

import "gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"

// AgentToken is agentk's bearer access token.
type AgentToken string

// ClientAgentMeta contains information about agentk that it sends on each request as gRPC metadata.
type ClientAgentMeta struct {
	Version      string
	CommitId     string
	PodNamespace string
	PodName      string
}

// AgentMeta contains information received from agentk with a request.
// It's passed as gRPC metadata.
type AgentMeta struct {
	Token AgentToken
	ClientAgentMeta
}

type GitalyInfo struct {
	Address  string
	Token    string
	Features map[string]string
}

// AgentInfo contains information about an agentk.
type AgentInfo struct {
	// Id is the agent's id in the database.
	Id int64
	// ProjectId is the id of the configuration project of the agent.
	ProjectId int64

	// Name is the agent's name.
	// Can contain only /a-z\d-/
	Name       string
	GitalyInfo GitalyInfo
	Repository gitalypb.Repository
}

type ProjectInfo struct {
	ProjectId  int64
	GitalyInfo GitalyInfo
	Repository gitalypb.Repository
}
