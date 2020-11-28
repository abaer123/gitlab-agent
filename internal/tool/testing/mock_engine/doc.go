// Mocks for gitops-engine.
package mock_engine

//go:generate go run github.com/golang/mock/mockgen -destination "engine.go" "github.com/argoproj/gitops-engine/pkg/engine" "GitOpsEngine"
//go:generate go run github.com/golang/mock/mockgen -destination "engine_factory.go" -package "mock_engine" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/agent" "GitOpsEngineFactory"
