package agent

//go:generate go run github.com/golang/mock/mockgen -destination "mock_for_observer_test.go" -package "agent" "github.com/cilium/cilium/api/v1/observer" "ObserverClient,Observer_GetFlowsClient"
