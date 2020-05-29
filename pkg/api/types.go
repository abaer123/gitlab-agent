package api

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
	// Name is agent's name.
	// Can contain only /a-z\d-/
	Name       string
	Repository AgentConfigRepository
}

// AgentConfigRepository represents agentk's configuration repository.
type AgentConfigRepository struct {
	StorageName   string
	RelativePath  string
	GlRepository  string
	GlProjectPath string
}
