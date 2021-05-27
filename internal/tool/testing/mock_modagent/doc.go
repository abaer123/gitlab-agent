package mock_modagent

//go:generate go run github.com/golang/mock/mockgen -destination "api.go" -package "mock_modagent" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modagent" "API,Factory,Module"
