package mock_redis

//go:generate go run github.com/golang/mock/mockgen -destination "expiring_hash.go" -package "mock_redis" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/redistool" "ExpiringHashInterface"
