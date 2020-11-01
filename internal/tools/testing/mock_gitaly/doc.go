// Mocks for Gitaly.
package mock_gitaly

//go:generate go run github.com/golang/mock/mockgen -destination "gitaly.go" -package "mock_gitaly" "gitlab.com/gitlab-org/gitaly/proto/go/gitalypb" "CommitServiceClient,CommitService_TreeEntryClient,SmartHTTPServiceClient,SmartHTTPService_InfoRefsUploadPackClient"

//go:generate go run github.com/golang/mock/mockgen -destination "internal_gitaly.go" -package "mock_gitaly" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly" "PoolInterface"
