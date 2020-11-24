package mock_modclient

//go:generate go run github.com/golang/mock/mockgen -destination "api.go" -package "mock_modclient" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/modules/modclient" "AgentAPI,Factory,Module"
