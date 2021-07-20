package api

import (
	"context"
	"net/http"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
)

const (
	UsagePingApiPath = "/api/v4/internal/kubernetes/usage_metrics"
)

func SendUsagePing(ctx context.Context, client gitlab.ClientInterface, counters map[string]int64) error {
	return client.Do(ctx,
		gitlab.WithMethod(http.MethodPost),
		gitlab.WithPath(UsagePingApiPath),
		gitlab.WithJsonRequestBody(counters),
		gitlab.WithResponseHandler(gitlab.NoContentResponseHandler()),
		gitlab.WithJWT(true),
	)
}
