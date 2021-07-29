package api

import (
	"context"
	"net/http"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	AgentConfigurationApiPath = "/api/v4/internal/kubernetes/agent_configuration"
)

type agentConfigurationRequest struct {
	AgentId     int64             `json:"agent_id"`
	AgentConfig *agentConfigAlias `json:"agent_config"`
}

func PostAgentConfiguration(ctx context.Context, client gitlab.ClientInterface, agentId int64, config *agentcfg.ConfigurationFile) error {
	return client.Do(ctx,
		gitlab.WithMethod(http.MethodPost),
		gitlab.WithPath(AgentConfigurationApiPath),
		gitlab.WithJWT(true),
		gitlab.WithJsonRequestBody(agentConfigurationRequest{
			AgentId:     agentId,
			AgentConfig: (*agentConfigAlias)(config),
		}),
		gitlab.WithResponseHandler(gitlab.NoContentResponseHandler()),
	)
}

// agentConfigAlias ensures the protojson package is used for to/from JSON marshaling.
// See https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson.
type agentConfigAlias agentcfg.ConfigurationFile

func (c *agentConfigAlias) MarshalJSON() ([]byte, error) {
	typedC := (*agentcfg.ConfigurationFile)(c)
	return protojson.Marshal(typedC)
}

func (c *agentConfigAlias) UnmarshalJSON(data []byte) error {
	typedC := (*agentcfg.ConfigurationFile)(c)
	return protojson.Unmarshal(data, typedC)
}
