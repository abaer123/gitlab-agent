package kascfg_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd/kas/kasapp"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/kascfg"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"sigs.k8s.io/yaml"
)

const (
	kasConfigExampleFile = "config_example.yaml"
)

func TestExampleConfigHasCorrectDefaults(t *testing.T) {
	// This is effectively the minimum required configuration i.e. only the required fields.
	cfgDefaulted := &kascfg.ConfigurationFile{
		Gitlab: &kascfg.GitLabCF{
			Address:                  "http://localhost:8080",
			AuthenticationSecretFile: "/some/file",
		},
		Agent: &kascfg.AgentCF{
			KubernetesApi: &kascfg.KubernetesApiCF{
				Listen: &kascfg.ListenKubernetesApiCF{},
			},
		},
		// Not actually required, but Redis.RedisConfig.Server.Address is required if Redis key is specified so add it here to show that and defaults too.
		Redis: &kascfg.RedisCF{
			RedisConfig: &kascfg.RedisCF_Server{
				Server: &kascfg.RedisServerCF{
					Address: "localhost:6380",
				},
			},
			PasswordFile: "/some/file",
			Network:      "tcp",
		},
		// Not actually required, but Listen.AuthenticationSecretFile is required if Api key is specified so add it here to show that and defaults too.
		Api: &kascfg.ApiCF{
			Listen: &kascfg.ListenApiCF{
				AuthenticationSecretFile: "/some/file",
			},
		},
	}
	kasapp.ApplyDefaultsToKasConfigurationFile(cfgDefaulted)

	cfgFromFile, err := kasapp.LoadConfigurationFile(kasConfigExampleFile)
	if assert.NoError(t, err) {
		assert.Empty(t, cmp.Diff(cfgDefaulted, cfgFromFile, protocmp.Transform()))
	} else {
		// Failed to load. Just print what it should be
		data, err := protojson.Marshal(cfgDefaulted)
		require.NoError(t, err)
		configYAML, err := yaml.JSONToYAML(data)
		require.NoError(t, err)
		fmt.Println(string(configYAML)) // nolint: forbidigo
	}
}
