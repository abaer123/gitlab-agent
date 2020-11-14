package mock_internalgitaly

//go:generate go run github.com/golang/mock/mockgen -destination "internalgitaly.go" -package "mock_internalgitaly" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly" "PoolInterface,FetchVisitor,PathEntryVisitor,PathFetcherInterface,PollerInterface"
