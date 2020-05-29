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
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd/kgb/kgbapp"
	"golang.org/x/sync/errgroup"
)

// TestFetchConfiguration tests agentk's ability to fetch configuration from a repository.
// Flow:
// 1. agentk connections to kgb and asks for configuration, providing an access token
// 2. kgb makes a request to GitLab using that access token to verify the token and fetch information about the agent
// 3. kgb makes a request to Gitaly to fetch configuration, parses it and sends it back to agentk.
func TestFetchConfiguration(t *testing.T) {
	t.Run("Plain gRPC", func(t *testing.T) {
		testFetchConfiguration(t, false)
	})
	t.Run("gRPC->WebSocket", func(t *testing.T) {
		testFetchConfiguration(t, true)
	})
}

func testFetchConfiguration(t *testing.T, websocket bool) {
	gitalyAddress := getGitalyAddress(t)
	gitlabAddress := getGitLabAddress(t)
	address := getRandomLocalAddress(t)
	ag := kgbapp.App{
		ListenNetwork:             "tcp",
		ListenAddress:             address,
		ListenWebSocket:           websocket,
		GitalyAddress:             gitalyAddress,
		GitLabAddress:             gitlabAddress,
		ReloadConfigurationPeriod: 10 * time.Minute,
	}
	if websocket {
		address = "ws://" + address
	} else {
		address = "tcp://" + address
	}
	tokenFile := filepath.Join(os.TempDir(), fmt.Sprintf("%d.token", rand.Uint64()))
	t.Cleanup(func() {
		os.Remove(tokenFile)
	})
	require.NoError(t, ioutil.WriteFile(tokenFile, []byte("very secret token :D"), 0o644))
	ak := agentkapp.App{
		KgbAddress: address,
		Insecure:   true,
		TokenFile:  tokenFile,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)
	defer func() {
		assert.NoError(t, g.Wait())
	}()
	g.Go(func() error {
		return ag.Run(ctx)
	})
	time.Sleep(1 * time.Second) // let kgb start listening
	g.Go(func() error {
		return ak.Run(ctx)
	})

	// TODO
}

func getGitalyAddress(t *testing.T) string {
	gitalyAddress := os.Getenv("GITALY_ADDRESS")
	if gitalyAddress == "" {
		t.Skip(`Please set GITALY_ADDRESS="..." to run this integration test`)
		panic("unreachable") // this is never executed actually
	}
	return gitalyAddress
}

func getGitLabAddress(t *testing.T) string {
	gitLabAddress := os.Getenv("GITLAB_ADDRESS")
	if gitLabAddress == "" {
		t.Skip(`Please set GITLAB_ADDRESS="..." to run this integration test`)
		panic("unreachable") // this is never executed actually
	}
	return gitLabAddress
}

func getRandomLocalAddress(t *testing.T) string {
	l, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().String()
}
