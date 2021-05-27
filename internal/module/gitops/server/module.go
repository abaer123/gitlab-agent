package server

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type module struct {
	rpc.UnimplementedGitopsServer
	api                         modserver.API
	gitalyPool                  gitaly.PoolInterface
	projectInfoClient           *projectInfoClient
	syncCount                   usage_metrics.Counter
	gitOpsPollIntervalHistogram prometheus.Histogram
	getObjectsBackoff           retry.BackoffManagerFactory
	pollPeriod                  time.Duration
	maxConnectionAge            time.Duration
	maxManifestFileSize         int64
	maxTotalManifestFileSize    int64
	maxNumberOfPaths            uint32
	maxNumberOfFiles            uint32
}

func (m *module) Run(ctx context.Context) error {
	return nil
}

func (m *module) GetObjectsToSynchronize(req *rpc.ObjectsToSynchronizeRequest, server rpc.Gitops_GetObjectsToSynchronizeServer) error {
	ctx := server.Context()
	agentToken := api.AgentTokenFromContext(ctx)
	log := grpctool.LoggerFromContext(ctx)
	backoff := m.getObjectsBackoff()
	agentInfo, err := m.api.GetAgentInfo(ctx, log, agentToken)
	if err != nil {
		return err // no wrap
	}
	err = m.validateGetObjectsToSynchronizeRequest(req)
	if err != nil {
		return err // no wrap
	}
	p := pollJob{
		log:                         log.With(logz.AgentId(agentInfo.Id), logz.ProjectId(req.ProjectId)),
		api:                         m.api,
		gitalyPool:                  m.gitalyPool,
		projectInfoClient:           m.projectInfoClient,
		syncCount:                   m.syncCount,
		req:                         req,
		server:                      server,
		agentToken:                  agentToken,
		gitOpsPollIntervalHistogram: m.gitOpsPollIntervalHistogram,
		maxManifestFileSize:         m.maxManifestFileSize,
		maxTotalManifestFileSize:    m.maxTotalManifestFileSize,
		maxNumberOfFiles:            m.maxNumberOfFiles,
	}
	return m.api.PollWithBackoff(ctx, backoff, true, m.maxConnectionAge, m.pollPeriod, p.Attempt)
}

func (m *module) Name() string {
	return gitops.ModuleName
}

func (m *module) validateGetObjectsToSynchronizeRequest(req *rpc.ObjectsToSynchronizeRequest) error {
	numberOfPaths := uint32(len(req.Paths))
	if numberOfPaths > m.maxNumberOfPaths {
		// TODO validate config in GetConfiguration too and send it somewhere the user can see it https://gitlab.com/gitlab-org/gitlab/-/issues/277323
		// This check must be here, but there too.
		return status.Errorf(codes.InvalidArgument, "maximum number of GitOps paths per manifest project is %d, but %d was requested", m.maxNumberOfPaths, numberOfPaths)
	}
	return nil
}
