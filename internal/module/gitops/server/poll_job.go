package server

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/gitops/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
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
	globPrefix = regexp.MustCompile(`^([^\\*?[\]{}]+)/(.*)$`)
)

type pollJob struct {
	log                         *zap.Logger
	api                         modserver.API
	gitalyPool                  gitaly.PoolInterface
	projectInfoClient           *projectInfoClient
	syncCount                   usage_metrics.Counter
	req                         *rpc.ObjectsToSynchronizeRequest
	server                      rpc.Gitops_GetObjectsToSynchronizeServer
	agentToken                  api.AgentToken
	gitOpsPollIntervalHistogram prometheus.Histogram
	lastPoll                    time.Time
	maxManifestFileSize         int64
	maxTotalManifestFileSize    int64
	maxNumberOfFiles            uint32
}

func (j *pollJob) Attempt() (error, retry.AttemptResult) {
	ctx := j.server.Context()
	// This call is made on each poll because:
	// - it checks that the agent's token is still valid
	// - repository location in Gitaly might have changed
	projectInfo, err := j.getProjectInfo(ctx, j.log, j.agentToken, j.req.ProjectId)
	if err != nil {
		return err, retry.Done // no wrap
	}
	if projectInfo == nil { // retriable error
		return nil, retry.Backoff
	}
	revision := gitaly.DefaultBranch // TODO support user-specified branches/tags
	p, err := j.gitalyPool.Poller(ctx, &projectInfo.GitalyInfo)
	if err != nil {
		j.api.HandleProcessingError(ctx, j.log, "GitOps: Poller", err)
		return nil, retry.Backoff
	}
	info, err := p.Poll(ctx, &projectInfo.Repository, j.req.CommitId, revision)
	if err != nil {
		j.api.HandleProcessingError(ctx, j.log, "GitOps: repository poll failed", err)
		return nil, retry.Backoff
	}

	j.trackPollInterval()

	if !info.UpdateAvailable {
		j.log.Debug("GitOps: no updates", logz.CommitId(j.req.CommitId))
		return nil, retry.Continue
	}
	log := j.log.With(logz.CommitId(info.CommitId))
	log.Info("GitOps: new commit")
	err = j.sendObjectsToSynchronizeHeader(log, info.CommitId)
	if err != nil {
		return err, retry.Done // no wrap
	}
	filesVisited, filesSent, err := j.sendObjectsToSynchronizeBody(log, j.req, &projectInfo.Repository, &projectInfo.GitalyInfo, info.CommitId)
	if err != nil {
		return err, retry.Done // no wrap
	}
	err = j.sendObjectsToSynchronizeTrailer(log)
	if err != nil {
		return err, retry.Done // no wrap
	}
	log.Info("GitOps: fetched files", logz.NumberOfFilesVisited(filesVisited), logz.NumberOfFilesSent(filesSent))
	j.syncCount.Inc()
	return nil, retry.Done
}

func (j *pollJob) trackPollInterval() {
	now := time.Now()

	if !j.lastPoll.IsZero() {
		pollInterval := now.Sub(j.lastPoll).Seconds()
		j.gitOpsPollIntervalHistogram.Observe(pollInterval)
	}

	j.lastPoll = now
}

func (j *pollJob) sendObjectsToSynchronizeHeader(log *zap.Logger, commitId string) error {
	err := j.server.Send(&rpc.ObjectsToSynchronizeResponse{
		Message: &rpc.ObjectsToSynchronizeResponse_Header_{
			Header: &rpc.ObjectsToSynchronizeResponse_Header{
				CommitId: commitId,
			},
		},
	})
	if err != nil {
		return j.api.HandleSendError(log, "GitOps: failed to send header for objects to synchronize", err)
	}
	return nil
}

func (j *pollJob) sendObjectsToSynchronizeBody(
	log *zap.Logger,
	req *rpc.ObjectsToSynchronizeRequest,
	repo *gitalypb.Repository,
	gitalyInfo *api.GitalyInfo,
	commitId string,
) (uint32 /* files visited */, uint32 /* files sent */, error) {
	ctx := j.server.Context()
	pf, err := j.gitalyPool.PathFetcher(ctx, gitalyInfo)
	if err != nil {
		j.api.HandleProcessingError(ctx, log, "GitOps: PathFetcher", err)
		return 0, 0, status.Error(codes.Unavailable, "GitOps: PathFetcher")
	}
	v := &objectsToSynchronizeVisitor{
		server:        j.server,
		fileSizeLimit: j.maxManifestFileSize,
	}
	var delegate gitaly.FetchVisitor = v
	delegate = gitaly.NewChunkingFetchVisitor(delegate, gitOpsManifestMaxChunkSize)
	delegate = gitaly.NewTotalSizeLimitingFetchVisitor(delegate, j.maxTotalManifestFileSize)
	delegate = gitaly.NewDuplicateFileDetectingVisitor(delegate)
	delegate = gitaly.NewHiddenDirFilteringFetchVisitor(delegate)
	vGlob := gitaly.NewGlobFilteringFetchVisitor(delegate, "")
	vCounting := gitaly.NewEntryCountLimitingFetchVisitor(vGlob, j.maxNumberOfFiles)
	for _, p := range req.Paths {
		globNoSlash := strings.TrimPrefix(p.Glob, "/") // original glob without the leading slash
		repoPath, recursive := globToGitaly(globNoSlash)
		vGlob.Glob = globNoSlash // set new glob for each path
		err = pf.Visit(ctx, repo, []byte(commitId), repoPath, recursive, vCounting)
		if err != nil {
			if v.sendFailed {
				return vCounting.FilesVisited, vCounting.FilesSent, j.api.HandleSendError(log, "GitOps: failed to send objects to synchronize", err)
			}
			if isUserError(err) {
				err = errz.NewUserErrorWithCause(err, "manifest file")
				j.api.HandleProcessingError(ctx, log, "GitOps: failed to get objects to synchronize", err)
				// return the error to the client because it's a user error
				return vCounting.FilesVisited, vCounting.FilesSent, status.Errorf(codes.FailedPrecondition, "GitOps: failed to get objects to synchronize: %v", err)
			}
			j.api.HandleProcessingError(ctx, log, "GitOps: failed to get objects to synchronize", err)
			return vCounting.FilesVisited, vCounting.FilesSent, status.Error(codes.Unavailable, "GitOps: failed to get objects to synchronize")
		}
	}
	return vCounting.FilesVisited, vCounting.FilesSent, nil
}

func (j *pollJob) sendObjectsToSynchronizeTrailer(log *zap.Logger) error {
	err := j.server.Send(&rpc.ObjectsToSynchronizeResponse{
		Message: &rpc.ObjectsToSynchronizeResponse_Trailer_{
			Trailer: &rpc.ObjectsToSynchronizeResponse_Trailer{},
		},
	})
	if err != nil {
		return j.api.HandleSendError(log, "GitOps: failed to send trailer for objects to synchronize", err)
	}
	return nil
}

// getProjectInfo returns nil for both error and ProjectInfo if there was a retriable error.
func (j *pollJob) getProjectInfo(ctx context.Context, log *zap.Logger, agentToken api.AgentToken, projectId string) (*api.ProjectInfo, error) {
	projectInfo, err := j.projectInfoClient.GetProjectInfo(ctx, agentToken, projectId)
	switch {
	case err == nil:
		return projectInfo, nil
	case errz.ContextDone(err):
		err = status.Error(codes.Unavailable, "unavailable")
	case gitlab.IsForbidden(err):
		err = status.Error(codes.PermissionDenied, "forbidden")
	case gitlab.IsUnauthorized(err):
		err = status.Error(codes.Unauthenticated, "unauthenticated")
	default:
		j.api.HandleProcessingError(ctx, log, "GetProjectInfo()", err)
		err = nil // no error and no project info
	}
	return nil, err
}

func isUserError(err error) bool {
	switch err.(type) { // nolint:errorlint
	case *gitaly.GlobMatchFailedError, *gitaly.MaxNumberOfFilesError, *gitaly.DuplicatePathFoundError:
		return true
	}
	switch gitaly.ErrorCodeFromError(err) { // nolint:exhaustive
	case gitaly.NotFound, gitaly.FileTooBig, gitaly.UnexpectedTreeEntryType:
		return true
	}
	return false
}

// globToGitaly accepts a glob without a leading slash!
func globToGitaly(glob string) ([]byte /* repoPath */, bool /* recursive */) {
	var repoPath []byte
	matches := globPrefix.FindStringSubmatch(glob)
	if matches == nil {
		repoPath = []byte{'.'}
	} else {
		repoPath = []byte(matches[1])
		glob = matches[2]
	}
	recursive := strings.ContainsAny(glob, "[/") || // cannot determine if recursive or not because character class may contain ranges, etc
		strings.Contains(glob, "**") // contains directory match
	return repoPath, recursive
}
