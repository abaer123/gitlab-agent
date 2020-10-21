package mock_grpc

//go:generate go run github.com/golang/mock/mockgen -destination "streamserver.go" -package "mock_grpc" "google.golang.org/grpc" "ServerStream"
