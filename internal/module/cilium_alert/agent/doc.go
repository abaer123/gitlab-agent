package agent

//go:generate go run github.com/golang/mock/mockgen -destination "mock_for_observer_test.go" -package "agent" "github.com/cilium/cilium/api/v1/observer" "ObserverClient,Observer_GetFlowsClient"
//go:generate go run github.com/golang/mock/mockgen -destination "mock_for_cilium_io_test.go" -package "agent" "github.com/cilium/cilium/pkg/k8s/client/clientset/versioned/typed/cilium.io/v2" "CiliumV2Interface,CiliumNetworkPolicyInterface"
