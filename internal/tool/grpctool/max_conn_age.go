package grpctool

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// a) below is the value of the max connection age parameter. It typically is the max connection age of a load balancer
// in front of kas. After it elapses, the load balancer aborts the connection and the client sees a connection reset.
// This is not good. To mitigate the problem all long running server-side poll loops should stop beforehand.
// modserver.API.PollWithBackoff(maxConnectionAge) should be used for that.
//
// After b1) / b2) time elapses, gRPC will send a HTTP/2 GOAWAY frame to the client, to let it know that it
// should not use this connection for any new RPCs and, after all in-flight requests had finished, close it.
// After b1) / b2) time elapses, max connection grace period starts, allowing in-flight requests to finish cleanly.
// After this second time interval elapses, gRPC aborts the connection.
// https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/138 is the example of the above
// behavior. To fix it we need to correctly calculate the following parameters based on the max connection age parameter:
// - gRPC max connection age
// - max poll duration

// "Idle" flow.
// In this situation b1 must be smaller than c1 so that kas sends a GOAWAY to the client before it
// tries to reuse the connection. That way a client can only make a single long running request over a TCP
// connection and then the connection will be closed.
// time -->
// a)  |<-- max connection age parameter value --------------------------------------->|
// b1) |<-- gRPC max connection age +10% -------------->|          // +10% of jitter edge case
// c1) |<-- r1 (a bit bigger than b) ------------------->|         // c1 is maxPollDuration here

// "Fast" first request+slow second request flow.
// In this situation b1 must be smaller than c1 so that kas sends a GOAWAY to the client before.
//// client tries to reuse the connection.
// time -->
// a)  |<-- max connection age parameter value --------------------------------------->|
// b2) |<-- gRPC max connection age -10% -------------->|          // -10% of jitter edge case
// c2) |<-- r1 (a bit smaller than b) --------------->|
// d2)                                                 |<-- r2 (max poll duration) -->|

// https://github.com/grpc/grpc-go/issues/4597 feature request to get rid of the function below.

func MaxConnectionAge2MaxPollDuration(maxConnectionAge time.Duration) time.Duration {
	maxConnectionAge = maxConnectionAge * 9 / 10 // -10%. Generous chunk of time left for grace period
	maxConnectionAge /= 2
	return maxConnectionAge
}

func MaxConnectionAge2GrpcKeepalive(maxConnectionAge time.Duration) grpc.ServerOption {
	// See https://github.com/grpc/grpc-go/blob/v1.33.1/internal/transport/http2_server.go#L949-L1047
	// to better understand how keepalive works.
	return grpc.KeepaliveParams(maxConnectionAge2GrpcKeepalive(maxConnectionAge))
}

func maxConnectionAge2GrpcKeepalive(maxConnectionAge time.Duration) keepalive.ServerParameters {
	return keepalive.ServerParameters{
		// maxConnectionAge / 2 (see the diagram above) - 10% (jitter) - 5% (slack) = 35%
		MaxConnectionAge: maxConnectionAge * 35 / 100,
		// Give pending RPCs plenty of time to complete.
		MaxConnectionAgeGrace: maxConnectionAge,
		// Trying to stay below 60 seconds (typical load-balancer timeout)
		Time: 50 * time.Second,
	}
}
