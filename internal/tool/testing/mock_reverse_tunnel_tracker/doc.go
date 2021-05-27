package mock_reverse_tunnel_tracker

//go:generate go run github.com/golang/mock/mockgen -destination "tracker.go" -package "mock_reverse_tunnel_tracker" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel/tracker" "Registerer,Querier"
