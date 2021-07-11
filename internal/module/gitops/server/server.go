package server

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
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

func (s *server) GetObjectsToSynchronize(req *rpc.ObjectsToSynchronizeRequest, server rpc.Gitops_GetObjectsToSynchronizeServer) error {
	ctx := server.Context()
	agentToken := api.AgentTokenFromContext(ctx)
	log := grpctool.LoggerFromContext(ctx)
	backoff := s.getObjectsBackoff()
	agentInfo, err := s.api.GetAgentInfo(ctx, log, agentToken)
	if err != nil {
		return err // no wrap
	}
	err = s.validateGetObjectsToSynchronizeRequest(req)
	if err != nil {
		return err // no wrap
	}
	p := pollJob{
		log:                         log.With(logz.AgentId(agentInfo.Id), logz.ProjectId(req.ProjectId)),
		api:                         s.api,
		gitalyPool:                  s.gitalyPool,
		projectInfoClient:           s.projectInfoClient,
		syncCount:                   s.syncCount,
		req:                         req,
		server:                      server,
		agentToken:                  agentToken,
		gitOpsPollIntervalHistogram: s.gitOpsPollIntervalHistogram,
		maxManifestFileSize:         s.maxManifestFileSize,
		maxTotalManifestFileSize:    s.maxTotalManifestFileSize,
		maxNumberOfFiles:            s.maxNumberOfFiles,
	}
	return s.api.PollWithBackoff(ctx, backoff, true, s.maxConnectionAge, s.pollPeriod, p.Attempt)
}

func (s *server) validateGetObjectsToSynchronizeRequest(req *rpc.ObjectsToSynchronizeRequest) error {
	numberOfPaths := uint32(len(req.Paths))
	if numberOfPaths > s.maxNumberOfPaths {
		// TODO validate config in GetConfiguration too and send it somewhere the user can see it https://gitlab.com/gitlab-org/gitlab/-/issues/277323
		// This check must be here, but there too.
		return status.Errorf(codes.InvalidArgument, "maximum number of GitOps paths per manifest project is %d, but %d was requested", s.maxNumberOfPaths, numberOfPaths)
	}
	return nil
}
