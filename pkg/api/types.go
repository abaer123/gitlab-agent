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

type GitalyInfo struct {
	Address  string
	Token    string
	Features map[string]string
}

// AgentInfo contains information about an agentk.
type AgentInfo struct {
	Meta AgentMeta
	// ID is the agent's id in the database.
	ID int64

	// Name is the agent's name.
	// Can contain only /a-z\d-/
	Name       string
	GitalyInfo GitalyInfo
	Repository gitalypb.Repository
}

type ProjectInfo struct {
	ProjectID  int64
	GitalyInfo GitalyInfo
	Repository gitalypb.Repository
}
