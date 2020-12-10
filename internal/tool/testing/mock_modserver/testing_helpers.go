package mock_modserver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/grpctool"
	"google.golang.org/grpc/metadata"
)

func IncomingCtx(ctx context.Context, t *testing.T, agentToken api.AgentToken) context.Context {
	creds := grpctool.NewTokenCredentials(agentToken, false)
	meta, err := creds.GetRequestMetadata(context.Background())
	require.NoError(t, err)
	ctx = metadata.NewIncomingContext(ctx, metadata.New(meta))
	agentMD, err := grpctool.AgentMDFromRawContext(ctx)
	require.NoError(t, err)
	return api.InjectAgentMD(ctx, agentMD)
}
