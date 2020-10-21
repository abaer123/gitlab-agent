package mock_redis

//go:generate go run github.com/golang/mock/mockgen -destination "conn.go" -package "mock_redis" "github.com/gomodule/redigo/redis" "Conn"
//go:generate go run github.com/golang/mock/mockgen -destination "pool.go" -package "mock_redis" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/redis" "Pool"
