package it

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd/agentk/agentkapp"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd/kgb/kgbapp"
	"golang.org/x/sync/errgroup"
)

// TestIntegration starts agentg, agentk and a Kubernetes API mock server.
// Then the test makes a request to agentg mimicking GitLab.
// Expectations:
// - agentk should connect to agentg
// - agentg should accept connection from the test
// - test should be able to make a request to agentg and get a response from Kubernetes API mock server.
func Test(t *testing.T) {
	t.Run("Plain gRPC", func(t *testing.T) {
		testFetchConfiguration(t, false)
	})
	t.Run("gRPC->WebSocket", func(t *testing.T) {
		testFetchConfiguration(t, true)
	})
}

func testFetchConfiguration(t *testing.T, websocket bool) {
	address := "localhost:12323"
	ag := kgbapp.App{
		ListenNetwork:             "tcp",
		ListenAddress:             address,
		ListenWebSocket:           websocket,
		GitalyAddress:             "unix:/Users/mikhail/src/gitlab-development-kit/praefect.socket",
		ReloadConfigurationPeriod: 10 * time.Minute,
	}
	if websocket {
		address = "ws://" + address
	}
	ak := agentkapp.App{
		KgbAddress: address,
		Insecure:   true,
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
