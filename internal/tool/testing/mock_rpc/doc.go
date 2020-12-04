// Mocks for gRPC services.
package mock_rpc

//go:generate go run github.com/golang/mock/mockgen -destination "grpc.go" -package "mock_rpc" "google.golang.org/grpc" "ServerStream,ClientStream,ClientConnInterface"

//go:generate go run github.com/golang/mock/mockgen -destination "gitops.go" -package "mock_rpc" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc" "GitopsClient,Gitops_GetObjectsToSynchronizeClient,Gitops_GetObjectsToSynchronizeServer,ObjectsToSynchronizeWatcherInterface"

//go:generate go run github.com/golang/mock/mockgen -destination "agent_configuration.go" -package "mock_rpc" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc" "AgentConfigurationClient,AgentConfiguration_GetConfigurationClient,AgentConfiguration_GetConfigurationServer,ConfigurationWatcherInterface"
