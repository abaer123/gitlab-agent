package server

import (
	"context"
	"regexp"
	"strings"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	gitOpsManifestMaxChunkSize = 128 * 1024
)

var (
	// globPrefix captures glob prefix that does not contain any special characters, recognized by doublestar.Match.
	// See https://github.com/bmatcuk/doublestar#about and
	// https://pkg.go.dev/github.com/bmatcuk/doublestar/v2#Match for globbing rules.
	globPrefix = regexp.MustCompile(`^/?([^\\*?[\]{}]+)/(.*)$`)
)

type pollJob struct {
	ctx                      context.Context
	log                      *zap.Logger
	api                      modserver.API
	gitalyPool               gitaly.PoolInterface
	projectInfoClient        *projectInfoClient
	syncCount                usage_metrics.Counter
	req                      *rpc.ObjectsToSynchronizeRequest
	server                   rpc.Gitops_GetObjectsToSynchronizeServer
	agentToken               api.AgentToken
	maxManifestFileSize      int64
	maxTotalManifestFileSize int64
	maxNumberOfFiles         uint32
}

func (j *pollJob) Attempt() (bool /*done*/, error) {
	// This call is made on each poll because:
	// - it checks that the agent's token is still valid
	// - repository location in Gitaly might have changed
	projectInfo, err, retErr := j.getProjectInfo(j.ctx, j.log, j.agentToken, j.req.ProjectId)
	if retErr {
		return false, err
	}
	revision := gitaly.DefaultBranch // TODO support user-specified branches/tags
	p, err := j.gitalyPool.Poller(j.ctx, &projectInfo.GitalyInfo)
	if err != nil {
		j.api.HandleProcessingError(j.ctx, j.log, "GitOps: Poller", err)
		return false, nil // don't want to close the response stream, so report no error
	}
	info, err := p.Poll(j.ctx, &projectInfo.Repository, j.req.CommitId, revision)
	if err != nil {
		j.api.HandleProcessingError(j.ctx, j.log, "GitOps: repository poll failed", err)
		return false, nil // don't want to close the response stream, so report no error
	}
	if !info.UpdateAvailable {
		j.log.Debug("GitOps: no updates", logz.CommitId(j.req.CommitId))
		return false, nil
	}
	log := j.log.With(logz.CommitId(info.CommitId))
	log.Info("GitOps: new commit")
	err = j.sendObjectsToSynchronizeHeaders(j.server, log, info.CommitId)
	if err != nil {
		return false, err // no wrap
	}
	numberOfFiles, err := j.sendObjectsToSynchronizeBody(j.req, j.server, log, &projectInfo.Repository, &projectInfo.GitalyInfo, info.CommitId)
	if err != nil {
		return false, err // no wrap
	}
	err = j.sendObjectsToSynchronizeTrailers(j.server, log)
	if err != nil {
		return false, err // no wrap
	}
	log.Info("GitOps: fetched files", logz.NumberOfFiles(numberOfFiles))
	j.syncCount.Inc()
	return true, nil
}

func (j *pollJob) sendObjectsToSynchronizeHeaders(server rpc.Gitops_GetObjectsToSynchronizeServer, log *zap.Logger, commitId string) error {
	err := server.Send(&rpc.ObjectsToSynchronizeResponse{
		Message: &rpc.ObjectsToSynchronizeResponse_Headers_{
			Headers: &rpc.ObjectsToSynchronizeResponse_Headers{
				CommitId: commitId,
			},
		},
	})
	if err != nil {
		return j.api.HandleSendError(log, "GitOps: failed to send headers for objects to synchronize", err)
	}
	return nil
}

func (j *pollJob) sendObjectsToSynchronizeBody(req *rpc.ObjectsToSynchronizeRequest, server rpc.Gitops_GetObjectsToSynchronizeServer, log *zap.Logger, repo *gitalypb.Repository, gitalyInfo *api.GitalyInfo, commitId string) (uint32, error) {
	ctx := server.Context()
	pf, err := j.gitalyPool.PathFetcher(ctx, gitalyInfo)
	if err != nil {
		j.api.HandleProcessingError(ctx, log, "GitOps: PathFetcher", err)
		return 0, status.Error(codes.Unavailable, "GitOps: PathFetcher")
	}
	v := &objectsToSynchronizeVisitor{
		server:                 server,
		remainingTotalFileSize: j.maxTotalManifestFileSize,
		fileSizeLimit:          j.maxManifestFileSize,
		maxNumberOfFiles:       j.maxNumberOfFiles,
	}
	vChunk := gitaly.ChunkingFetchVisitor{
		MaxChunkSize: gitOpsManifestMaxChunkSize,
		Delegate:     v,
	}
	for _, p := range req.Paths {
		repoPath, recursive, glob := globToGitaly(p.Glob)
		v.glob = glob // set new glob for each path
		err = pf.Visit(ctx, repo, []byte(commitId), repoPath, recursive, vChunk)
		if err != nil {
			if v.sendFailed {
				return 0, j.api.HandleSendError(log, "GitOps: failed to send objects to synchronize", err)
			} else {
				j.api.HandleProcessingError(ctx, log, "GitOps: failed to get objects to synchronize", err)
				return 0, status.Error(codes.Unavailable, "GitOps: failed to get objects to synchronize")
			}
		}
	}
	return v.numberOfFiles, nil
}

func (j *pollJob) sendObjectsToSynchronizeTrailers(server rpc.Gitops_GetObjectsToSynchronizeServer, log *zap.Logger) error {
	err := server.Send(&rpc.ObjectsToSynchronizeResponse{
		Message: &rpc.ObjectsToSynchronizeResponse_Trailers_{
			Trailers: &rpc.ObjectsToSynchronizeResponse_Trailers{},
		},
	})
	if err != nil {
		return j.api.HandleSendError(log, "GitOps: failed to send trailers for objects to synchronize", err)
	}
	return nil
}

func (j *pollJob) getProjectInfo(ctx context.Context, log *zap.Logger, agentToken api.AgentToken, projectId string) (*api.ProjectInfo, error, bool /* return the error? */) {
	projectInfo, err := j.projectInfoClient.GetProjectInfo(ctx, agentToken, projectId)
	switch {
	case err == nil:
		return projectInfo, nil, false
	case errz.ContextDone(err):
		err = status.Error(codes.Unavailable, "unavailable")
	case gitlab.IsForbidden(err):
		err = status.Error(codes.PermissionDenied, "forbidden")
	case gitlab.IsUnauthorized(err):
		err = status.Error(codes.Unauthenticated, "unauthenticated")
	default:
		j.api.HandleProcessingError(ctx, log, "GetProjectInfo()", err)
		err = nil // don't want to close the response stream, so report no error
	}
	return nil, err, true
}

func globToGitaly(glob string) ([]byte /* repoPath */, bool /* recursive */, string /* glob */) {
	var repoPath []byte
	matches := globPrefix.FindStringSubmatch(glob)
	if matches == nil {
		repoPath = []byte{'.'}
		glob = strings.TrimPrefix(glob, "/") // remove at most one slash to match regex
	} else {
		repoPath = []byte(matches[1])
		glob = matches[2]
	}
	recursive := strings.ContainsAny(glob, "[/") || // cannot determine if recursive or not because character class may contain ranges, etc
		strings.Contains(glob, "**") // contains directory match
	return repoPath, recursive, glob
}
