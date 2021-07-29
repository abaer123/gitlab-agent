package api

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/testhelpers"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"google.golang.org/protobuf/testing/protocmp"
)

var (
	_ json.Marshaler   = (*agentConfigAlias)(nil)
	_ json.Unmarshaler = (*agentConfigAlias)(nil)
)

func TestPostAgentConfiguration(t *testing.T) {
	config := &agentcfg.ConfigurationFile{
		Gitops: &agentcfg.GitopsCF{
			ManifestProjects: []*agentcfg.ManifestProjectCF{
				{
					Id: "bla",
				},
			},
		},
		// don't need to test all fields, some is good enough
	}
	ctx, correlationId := testhelpers.CtxWithCorrelation(t)
	c := mock_gitlab.SetupClient(t, AgentConfigurationApiPath, func(w http.ResponseWriter, r *http.Request) {
		testhelpers.AssertJWTSignature(t, r)
		testhelpers.AssertRequestMethod(t, r, http.MethodPost)
		testhelpers.AssertRequestContentTypeJson(t, r)
		testhelpers.AssertCommonRequestParams(t, r, correlationId)
		data, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			return
		}
		var actual agentConfigurationRequest
		err = json.Unmarshal(data, &actual)
		if !assert.NoError(t, err) {
			return
		}
		expected := agentConfigurationRequest{
			AgentId:     testhelpers.AgentId,
			AgentConfig: (*agentConfigAlias)(config),
		}
		assert.Empty(t, cmp.Diff(expected, actual, cmp.Transformer("config", agentConfigAlias2ConfigurationFile), protocmp.Transform()))
		w.WriteHeader(http.StatusNoContent)
	})
	err := PostAgentConfiguration(ctx, c, testhelpers.AgentId, config)
	require.NoError(t, err)
}

func agentConfigAlias2ConfigurationFile(val *agentConfigAlias) *agentcfg.ConfigurationFile {
	return (*agentcfg.ConfigurationFile)(val)
}
