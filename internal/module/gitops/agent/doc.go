package agent

//go:generate go run github.com/golang/mock/mockgen -destination "mock_for_engine_test.go" -package "agent" "github.com/argoproj/gitops-engine/pkg/engine" "GitOpsEngine"
//go:generate go run github.com/golang/mock/mockgen -self_package "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/agent" -destination "mock_for_test.go" -package "agent" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/agent" "GitopsEngineFactory,GitopsWorkerFactory,GitopsWorker"
