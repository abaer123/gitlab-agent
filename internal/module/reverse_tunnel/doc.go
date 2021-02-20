package reverse_tunnel

//go:generate go run github.com/golang/mock/mockgen -self_package "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel" -destination "mock_for_test.go" -package "reverse_tunnel" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel" "TunnelDataCallback"
