package mock_gitalypool

//go:generate go run github.com/golang/mock/mockgen -destination "pool.go" -package "mock_gitalypool" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly" "PoolInterface"
