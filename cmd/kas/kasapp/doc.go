package kasapp

//go:generate go run github.com/golang/mock/mockgen  -destination "mock_for_test.go" -package "kasapp" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd/kas/kasapp" "KasPool,ClientConnInterface"