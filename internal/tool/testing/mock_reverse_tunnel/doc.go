package mock_reverse_tunnel

//go:generate go run github.com/golang/mock/mockgen -destination "api.go" -package "mock_reverse_tunnel" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel" "TunnelConnectionHandler"

//go:generate go run github.com/golang/mock/mockgen -destination "rpc.go" -package "mock_reverse_tunnel" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/rpc" "ReverseTunnel_ConnectServer,ReverseTunnel_ConnectClient,ReverseTunnelClient"

//go:generate go run github.com/golang/mock/mockgen -destination "tracker.go" -package "mock_reverse_tunnel" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/tracker" "Registerer"
