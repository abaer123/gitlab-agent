package it

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.com/ash2k/gitlab-agent/cmd/agentg/agentgapp"
	"gitlab.com/ash2k/gitlab-agent/cmd/agentk/agentkapp"
	"golang.org/x/sync/errgroup"
)

// TestIntegration starts agentg, agentk and a Kubernetes API mock server.
// Then the test makes a request to agentg mimicking GitLab.
// Expectations:
// - agentk should connect to agentg
// - agentg should accept connection from the test
// - test should be able to make a request to agentg and get a response from Kubernetes API mock server.
func TestIntegration(t *testing.T) {
	socketAddr := filepath.Join(os.TempDir(), fmt.Sprintf("%d.sock", rand.Uint64()))
	t.Cleanup(func() {
		err := os.Remove(socketAddr)
		if err != nil {
			t.Log(err)
		}
	})
	ag := agentgapp.App{
		ListenNetwork: "unix",
		ListenAddress: socketAddr,
	}
	ak := agentkapp.App{
		AgentgAddress: fmt.Sprintf("unix:%s", socketAddr),
		Insecure:      true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)
	defer func() {
		//assert.NoError(t, g.Wait()) // TODO put this bank once the test is in place
		assert.NoError(t, nil) // to keep the dependency
		g.Wait()
	}()
	g.Go(func() error {
		return ag.Run(ctx)
	})
	g.Go(func() error {
		return ak.Run(ctx)
	})

	// TODO
}
