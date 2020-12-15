package mock_agent_tracker

//go:generate go run github.com/golang/mock/mockgen -destination "tracker.go" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker" "Tracker"
