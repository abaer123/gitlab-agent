package mock_errtracker

//go:generate go run github.com/golang/mock/mockgen -destination "errtracker.go" -package "mock_errtracker" "gitlab.com/gitlab-org/labkit/errortracking" "Tracker"
