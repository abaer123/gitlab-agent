// Mocks for GitLab client.
package mock_gitlab

//go:generate go run github.com/golang/mock/mockgen -destination "gitlab.go" -package "mock_gitlab" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab" "ClientInterface"
