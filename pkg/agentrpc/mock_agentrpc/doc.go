// Mocks for gRPC services.
package mock_agentrpc

//go:generate go run github.com/golang/mock/mockgen -destination "agentrpc.go" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc" "GitLabServiceClient,GitLabService_GetObjectsToSynchronizeClient,GitLabService_GetConfigurationClient"
