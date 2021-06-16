package server

import (
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"sigs.k8s.io/yaml"
)

func TestEmptyConfig(t *testing.T) {
	t.Run("comments", func(t *testing.T) {
		data := []byte(`
#gitops:
#  manifest_projects:
#  - id: "root/gitops-manifests"
#    paths:
#      - glob: "/bla/**"
`)
		assertEmpty(t, data)
	})
	t.Run("empty", func(t *testing.T) {
		data := []byte("")
		assertEmpty(t, data)
	})
	t.Run("newline", func(t *testing.T) {
		data := []byte("\n")
		assertEmpty(t, data)
	})
}

func assertEmpty(t *testing.T, data []byte) {
	config, err := parseYAMLToConfiguration(data)
	require.NoError(t, err)
	diff := cmp.Diff(config, &agentcfg.ConfigurationFile{}, protocmp.Transform())
	assert.Empty(t, diff)
}

func TestYAMLToConfigurationAndBack(t *testing.T) {
	testCases := []struct {
		given, expected string
	}{
		{
			given: `{}
`, // empty config
			expected: `{}
`,
		},
		{
			given: `gitops: {}
`,
			expected: `gitops: {}
`,
		},
		{
			given: `gitops:
  manifest_projects: []
`,
			expected: `gitops: {}
`, // empty slice is omitted
		},
		{
			expected: `gitops:
  manifest_projects:
  - id: gitlab-org/cluster-integration/gitlab-agent
`,
			given: `gitops:
  manifest_projects:
  - id: gitlab-org/cluster-integration/gitlab-agent
`,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			config, err := parseYAMLToConfiguration([]byte(tc.given))
			require.NoError(t, err)
			configJson, err := protojson.Marshal(config)
			require.NoError(t, err)
			configYaml, err := yaml.JSONToYAML(configJson)
			require.NoError(t, err)
			diff := cmp.Diff(tc.expected, string(configYaml))
			assert.Empty(t, diff)
		})
	}
}
