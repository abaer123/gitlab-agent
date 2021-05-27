package mock_reverse_tunnel

//go:generate go run github.com/golang/mock/mockgen -destination "api.go" -package "mock_reverse_tunnel" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel" "TunnelHandler,TunnelFinder,Tunnel"
