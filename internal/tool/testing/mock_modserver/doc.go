package mock_modserver

//go:generate go run github.com/golang/mock/mockgen -destination "api.go" -package "mock_modserver" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver" "API"
