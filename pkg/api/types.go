package api

import "gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"

const (
	MetadataAuthorization = "authorization"
	MetadataAgentkVersion = "agentk-version"
)

// AgentToken is agentk's bearer access token.
type AgentToken string

// AgentMeta contains information received from agentk with a request.
// It's passed as gRPC metadata.
type AgentMeta struct {
	Token   AgentToken
	Version string
}

// AgentInfo contains information about an agentk.
type AgentInfo struct {
	Meta AgentMeta
	// Id is the agent's id in the database.
	Id int64

	// Name is the agent's name.
	// Can contain only /a-z\d-/
	Name       string
	Repository gitalypb.Repository
}

type ProjectInfo struct {
	ProjectId  int64
	Repository gitalypb.Repository
}
