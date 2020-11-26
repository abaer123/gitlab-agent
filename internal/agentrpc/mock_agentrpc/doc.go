// Mocks for gRPC services.
package mock_agentrpc

//go:generate go run github.com/golang/mock/mockgen -destination "agentrpc.go" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc" "KasClient,Kas_GetObjectsToSynchronizeClient,Kas_GetConfigurationClient,Kas_GetConfigurationServer,Kas_GetObjectsToSynchronizeServer,ConfigurationWatcherInterface,ObjectsToSynchronizeWatcherInterface"
