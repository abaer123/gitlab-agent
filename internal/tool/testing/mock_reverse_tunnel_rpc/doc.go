package mock_reverse_tunnel_rpc

//go:generate go run github.com/golang/mock/mockgen -destination "rpc.go" -package "mock_reverse_tunnel_rpc" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel/rpc" "ReverseTunnel_ConnectServer,ReverseTunnel_ConnectClient,ReverseTunnelClient"
