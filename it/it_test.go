package it

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ash2k/stager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd/agentk/agentkapp"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd/kas/kasapp"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modshared"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
)

// TestFetchConfiguration tests agentk's ability to fetch configuration from a project.
// Flow:
// 1. agentk connects to kas and asks for configuration, providing an access token
// 2. kas makes a request to GitLab using that access token to verify the token and fetch information about the agent
// 3. kas makes a request to Gitaly to fetch configuration, parses it and sends it back to agentk.
func TestFetchConfiguration(t *testing.T) {
	t.Run("Plain gRPC", func(t *testing.T) {
		testFetchConfiguration(t, false)
	})
	//t.Run("gRPC->WebSocket", func(t *testing.T) {
	//	testFetchConfiguration(t, true)
	//})
}

func testFetchConfiguration(t *testing.T, websocket bool) {
	gitlabAddress := getGitLabAddress(t)
	agentkToken := getAgentkToken(t)
	kasAuthSecretFile := getKasAuthSecretFile(t)
	address := getRandomLocalAddress(t)
	ag := kasapp.ConfiguredApp{
		Log: zaptest.NewLogger(t),
		Configuration: &kascfg.ConfigurationFile{
			Gitlab: &kascfg.GitLabCF{
				Address:                  gitlabAddress,
				AuthenticationSecretFile: kasAuthSecretFile,
			},
			Agent: &kascfg.AgentCF{
				Listen: &kascfg.ListenAgentCF{
					Address:   address,
					Websocket: websocket,
				},
			},
		},
	}
	kasapp.ApplyDefaultsToKasConfigurationFile(ag.Configuration)
	if websocket {
		address = "ws://" + address
	} else {
		address = "grpc://" + address
	}
	agentkTokenFile := filepath.Join(os.TempDir(), fmt.Sprintf("agentk_token.%d.txt", rand.Uint64()))
	t.Cleanup(func() {
		os.Remove(agentkTokenFile)
	})
	require.NoError(t, ioutil.WriteFile(agentkTokenFile, []byte(agentkToken), 0o644))
	configFlags := genericclioptions.NewTestConfigFlags()
	configFlags.WithClientConfig(getKubeConfig())
	level := zap.NewAtomicLevelAt(zap.DebugLevel)
	ak := agentkapp.App{
		Log:      zaptest.NewLogger(t, zaptest.Level(level)),
		LogLevel: level,
		AgentMeta: &modshared.AgentMeta{
			Version:      cmd.Version,
			CommitId:     cmd.Commit,
			PodNamespace: "",
			PodName:      "",
		},
		KasAddress:      address,
		TokenFile:       agentkTokenFile,
		K8sClientGetter: configFlags,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st := stager.New()
	stage := st.NextStage()
	stage.Go(ag.Run)
	time.Sleep(1 * time.Second) // let kas start listening
	stage.Go(ak.Run)

	// TODO

	assert.NoError(t, st.Run(ctx))
}

func getGitLabAddress(t *testing.T) string {
	return getEnvString(t, "GITLAB_ADDRESS")
}

func getAgentkToken(t *testing.T) string {
	return getEnvString(t, "AGENTK_TOKEN")
}

func getKasAuthSecretFile(t *testing.T) string {
	return getEnvString(t, "KAS_GITLAB_AUTH_SECRET")
}

func getKubeConfig() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{
		CurrentContext: os.Getenv("KUBECONTEXT"),
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
}

func getEnvString(t *testing.T, envKey string) string {
	envVal, ok := os.LookupEnv(envKey)
	if !ok {
		t.Skipf(`Please set %s="..." to run this integration test`, envKey)
		panic("unreachable") // this is never executed actually
	}
	return envVal
}

func getRandomLocalAddress(t *testing.T) string {
	l, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().String()
}
