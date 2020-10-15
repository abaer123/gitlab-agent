package doc

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd/kas/kasapp"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"sigs.k8s.io/yaml"
)

const (
	kasConfigExampleFile = "kas_config_example.yaml"
)

func TestExampleConfigHasCorrectDefaults(t *testing.T) {
	cfgDefaulted := &kascfg.ConfigurationFile{}
	err := kasapp.ApplyDefaultsToKasConfigurationFile(cfgDefaulted)
	require.NoError(t, err)

	cfgFromFile, err := kasapp.LoadConfigurationFile(kasConfigExampleFile)
	if assert.NoError(t, err) {
		assert.Empty(t, cmp.Diff(cfgDefaulted, cfgFromFile, protocmp.Transform()))
	} else {
		// Failed to load. Just print what it should be
		data, err := protojson.Marshal(cfgDefaulted)
		require.NoError(t, err)
		configYAML, err := yaml.JSONToYAML(data)
		require.NoError(t, err)
		fmt.Println(string(configYAML))
	}
}
