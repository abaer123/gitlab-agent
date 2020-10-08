package doc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd/kas/kasapp"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"google.golang.org/protobuf/testing/protocmp"
)

const (
	kasConfigExampleFile = "kas_config_example.yaml"
)

func TestExampleConfigHasCorrectDefaults(t *testing.T) {
	cfgFromFile, err := kasapp.LoadConfigurationFile(kasConfigExampleFile)
	require.NoError(t, err)

	cfgDefaulted := &kascfg.ConfigurationFile{}
	err = kasapp.ApplyDefaultsToKasConfigurationFile(cfgDefaulted)
	require.NoError(t, err)

	assert.Empty(t, cmp.Diff(cfgDefaulted, cfgFromFile, protocmp.Transform()))
}
