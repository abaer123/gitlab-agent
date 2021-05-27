package mock_modserver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/metadata"
)

func IncomingCtx(ctx context.Context, t *testing.T, agentToken api.AgentToken) context.Context {
	creds := grpctool.NewTokenCredentials(agentToken, false)
	meta, err := creds.GetRequestMetadata(context.Background())
	require.NoError(t, err)
	ctx = metadata.NewIncomingContext(ctx, metadata.New(meta))
	agentMD, err := grpctool.AgentMDFromRawContext(ctx)
	require.NoError(t, err)
	ctx = api.InjectAgentMD(ctx, agentMD)
	ctx = grpctool.InjectLogger(ctx, zaptest.NewLogger(t))
	return ctx
}
