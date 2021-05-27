package mock_kubernetes_api

//go:generate go run github.com/golang/mock/mockgen -destination "rpc.go" -package "mock_kubernetes_api" "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/kubernetes_api/rpc" "KubernetesApiClient,KubernetesApi_MakeRequestClient"
