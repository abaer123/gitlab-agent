package mock_usage_metrics

//go:generate go run github.com/golang/mock/mockgen -destination "api.go" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/usage_metrics" "UsageTrackerInterface,Counter"
