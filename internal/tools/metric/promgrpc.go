package metric

import (
	"github.com/piotrkowalczuk/promgrpc/v4"
)

// ClientStatsHandler is a simplified promgrpc.ClientStatsHandler().
func ClientStatsHandler() *promgrpc.StatsHandler {
	return promgrpc.NewStatsHandler(
		promgrpc.NewClientRequestsInFlightStatsHandler(promgrpc.NewClientRequestsInFlightGaugeVec()),
	)
}

// ServerStatsHandler is a simplified promgrpc.ServerStatsHandler().
func ServerStatsHandler() *promgrpc.StatsHandler {
	return promgrpc.NewStatsHandler(
		promgrpc.NewServerRequestsInFlightStatsHandler(promgrpc.NewServerRequestsInFlightGaugeVec()),
	)
}
