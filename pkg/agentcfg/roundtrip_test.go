package agentcfg

import (
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestStructToJSONAndBack(t *testing.T) {
	testCases := []*ConfigurationFile{
		{}, // empty config
		{
			Deployments: &DeploymentsCF{}, // no ManifestProjects
		},
		{
			Deployments: &DeploymentsCF{
				ManifestProjects: []*ManifestProjectCF{}, // empty list of ManifestProjects
			},
		},
		{
			Deployments: &DeploymentsCF{
				ManifestProjects: []*ManifestProjectCF{
					{
						Id: "gitlab-org/cluster-integration/gitlab-agent",
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			data, err := protojson.Marshal(tc)
			require.NoError(t, err)
			tcCopy := &ConfigurationFile{}
			err = protojson.Unmarshal(data, tcCopy)
			require.NoError(t, err)
			diff := cmp.Diff(tc, tcCopy, protocmp.Transform())
			assert.Empty(t, diff)
		})
	}
}

func TestJSONToStructAndBack(t *testing.T) {
	testCases := []struct {
		given, expected string
	}{
		{
			given:    `{}`, // empty config
			expected: `{}`,
		},
		{
			given:    `{"deployments":{}}`,
			expected: `{"deployments":{}}`,
		},
		{
			given:    `{"deployments":{"manifest_projects":[]}}`,
			expected: `{"deployments":{}}`, // empty slice is omitted
		},
		{
			given:    `{"deployments":{"manifest_projects":[{"id":"gitlab-org/cluster-integration/gitlab-agent"}]}}`,
			expected: `{"deployments":{"manifest_projects":[{"id":"gitlab-org/cluster-integration/gitlab-agent"}]}}`,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			tcCopy := &ConfigurationFile{}
			err := protojson.Unmarshal([]byte(tc.given), tcCopy)
			require.NoError(t, err)
			data, err := protojson.Marshal(tcCopy)
			require.NoError(t, err)
			diff := cmp.Diff(tc.expected, string(data))
			assert.Empty(t, diff)
		})
	}
}