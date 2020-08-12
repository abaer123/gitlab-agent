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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd/agentk/agentkapp"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd/kas/kasapp"
	"golang.org/x/sync/errgroup"
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
	kasToken := getKasToken(t)
	address := getRandomLocalAddress(t)
	ag := kasapp.App{
		ListenNetwork:             "tcp",
		ListenAddress:             address,
		ListenWebSocket:           websocket,
		GitLabAddress:             gitlabAddress,
		ReloadConfigurationPeriod: 10 * time.Second,
	}
	if websocket {
		address = "ws://" + address
	} else {
		address = "grpc://" + address
	}
	tokenFile := filepath.Join(os.TempDir(), fmt.Sprintf("%d.token", rand.Uint64()))
	t.Cleanup(func() {
		os.Remove(tokenFile)
	})
	require.NoError(t, ioutil.WriteFile(tokenFile, []byte(kasToken), 0o644))
	configFlags := genericclioptions.NewTestConfigFlags()
	configFlags.WithClientConfig(getKubeConfig())
	ak := agentkapp.App{
		KasAddress:      address,
		TokenFile:       tokenFile,
		K8sClientGetter: configFlags,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)
	defer func() {
		assert.NoError(t, g.Wait())
	}()
	g.Go(func() error {
		return ag.Run(ctx)
	})
	time.Sleep(1 * time.Second) // let kas start listening
	g.Go(func() error {
		return ak.Run(ctx)
	})

	// TODO
}

func getGitLabAddress(t *testing.T) string {
	return getEnvString(t, "GITLAB_ADDRESS")
}

func getKasToken(t *testing.T) string {
	return getEnvString(t, "KAS_TOKEN")
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
