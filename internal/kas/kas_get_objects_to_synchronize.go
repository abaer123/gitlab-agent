package kas

import (
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v2"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api/apiutil"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/logz"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/wait"
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

func (s *Server) GetObjectsToSynchronize(req *agentrpc.ObjectsToSynchronizeRequest, stream agentrpc.Kas_GetObjectsToSynchronizeServer) error {
	ctx := stream.Context()
	agentMeta := apiutil.AgentMetaFromContext(ctx)
	agentInfo, err, retErr := s.getAgentInfo(ctx, agentMeta, false)
	if retErr {
		return err
	}
	numberOfPaths := uint32(len(req.Paths))
	if numberOfPaths > s.maxGitopsNumberOfPaths {
		return status.Errorf(codes.InvalidArgument, "maximum number of GitOps paths per manifest project is %d, but %d was requested", s.maxGitopsNumberOfPaths, numberOfPaths)
	}
	return s.pollImmediateUntil(stream.Context(), s.gitopsPollPeriod, s.sendObjectsToSynchronize(agentInfo, req, stream))
}

func (s *Server) sendObjectsToSynchronize(agentInfo *api.AgentInfo, req *agentrpc.ObjectsToSynchronizeRequest, stream agentrpc.Kas_GetObjectsToSynchronizeServer) wait.ConditionFunc {
	ctx := stream.Context()
	l := s.log.With(logz.AgentId(agentInfo.Id), logz.ProjectId(req.ProjectId))
	return func() (bool /*done*/, error) {
		// This call is made on each poll because:
		// - it checks that the agent's token is still valid
		// - repository location in Gitaly might have changed
		projectInfo, err, retErr := s.getProjectInfo(ctx, l, &agentInfo.Meta, req.ProjectId)
		if retErr {
			return false, err
		}
		revision := gitaly.DefaultBranch // TODO support user-specified branches/tags
		p, err := s.gitalyPool.Poller(ctx, &projectInfo.GitalyInfo)
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				l.Warn("GitOps: Poller", zap.Error(err))
			}
			return false, nil // don't want to close the response stream, so report no error
		}
		info, err := p.Poll(ctx, &projectInfo.Repository, req.CommitId, revision)
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				l.Warn("GitOps: repository poll failed", zap.Error(err))
			}
			return false, nil // don't want to close the response stream, so report no error
		}
		if !info.UpdateAvailable {
			l.Debug("GitOps: no updates", logz.CommitId(req.CommitId))
			return false, nil
		}
		// Create a new l variable, don't want to mutate the one from the outer scope
		l := l.With(logz.CommitId(info.CommitId)) // nolint:govet
		l.Info("GitOps: new commit")
		pf, err := s.gitalyPool.PathFetcher(ctx, &projectInfo.GitalyInfo)
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				l.Warn("GitOps: PathFetcher", zap.Error(err))
			}
			return false, nil // don't want to close the response stream, so report no error
		}
		err = stream.Send(&agentrpc.ObjectsToSynchronizeResponse{
			Message: &agentrpc.ObjectsToSynchronizeResponse_Headers_{
				Headers: &agentrpc.ObjectsToSynchronizeResponse_Headers{
					CommitId: info.CommitId,
				},
			},
		})
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				l.Warn("GitOps: failed to send objects to synchronize", zap.Error(err))
			}
			return false, status.Error(codes.Unavailable, "unavailable")
		}
		v := &objectsToSynchronizeVisitor{
			stream:                 stream,
			remainingTotalFileSize: s.maxGitopsTotalManifestFileSize,
			fileSizeLimit:          s.maxGitopsManifestFileSize,
			maxNumberOfFiles:       s.maxGitopsNumberOfFiles,
		}
		vChunk := gitaly.ChunkingFetchVisitor{
			MaxChunkSize: gitOpsManifestMaxChunkSize,
			Delegate:     v,
		}
		for _, p := range req.Paths {
			repoPath, recursive, glob := globToGitaly(p.Glob)
			v.glob = glob // set new glob for each path
			err = pf.Visit(ctx, &projectInfo.Repository, []byte(info.CommitId), repoPath, recursive, vChunk)
			if err != nil {
				if !grpctool.RequestCanceled(err) {
					l.Warn("GitOps: failed to get objects to synchronize", zap.Error(err))
				}
				return false, status.Error(codes.Unavailable, "GitOps: failed to get objects to synchronize")
			}
		}
		err = stream.Send(&agentrpc.ObjectsToSynchronizeResponse{
			Message: &agentrpc.ObjectsToSynchronizeResponse_Trailers_{
				Trailers: &agentrpc.ObjectsToSynchronizeResponse_Trailers{},
			},
		})
		if err != nil {
			if !grpctool.RequestCanceled(err) {
				l.Warn("GitOps: failed to send objects to synchronize", zap.Error(err))
			}
			return false, status.Error(codes.Unavailable, "unavailable")
		}
		l.Info("GitOps: fetched files", logz.NumberOfFiles(v.numberOfFiles))
		s.usageMetrics.IncGitopsSyncCount()
		return true, nil
	}
}

func (s *Server) getProjectInfo(ctx context.Context, log *zap.Logger, agentMeta *api.AgentMeta, projectId string) (*api.ProjectInfo, error, bool /* return the error? */) {
	projectInfo, err := s.gitLabClient.GetProjectInfo(ctx, agentMeta, projectId)
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
		log.Warn("GitOps: failed to get project info", zap.Error(err))
		err = nil // don't want to close the response stream, so report no error
	}
	return nil, err, true
}

type objectsToSynchronizeVisitor struct {
	stream                 agentrpc.Kas_GetObjectsToSynchronizeServer
	glob                   string
	remainingTotalFileSize int64
	fileSizeLimit          int64
	maxNumberOfFiles       uint32
	numberOfFiles          uint32
}

func (v *objectsToSynchronizeVisitor) Entry(entry *gitalypb.TreeEntry) (bool /* download? */, int64 /* max size */, error) {
	if v.numberOfFiles == v.maxNumberOfFiles {
		return false, 0, fmt.Errorf("maximum number of manifest files limit reached: %d", v.maxNumberOfFiles)
	}
	v.numberOfFiles++
	filename := string(entry.Path)
	if isHiddenDir(filename) {
		return false, 0, nil
	}
	shouldDownload, err := doublestar.Match(v.glob, filename)
	if err != nil {
		return false, 0, err
	}
	return shouldDownload, minInt64(v.remainingTotalFileSize, v.fileSizeLimit), nil
}

func (v *objectsToSynchronizeVisitor) StreamChunk(path []byte, data []byte) (bool /* done? */, error) {
	v.remainingTotalFileSize -= int64(len(data))
	if v.remainingTotalFileSize < 0 {
		// This should never happen because we told Gitaly the maximum file size that we'd like to get.
		// i.e. we should have gotten an error from Gitaly if file is bigger than the limit.
		return false, errors.New("unexpected negative remaining total file size")
	}
	return false, v.stream.Send(&agentrpc.ObjectsToSynchronizeResponse{
		Message: &agentrpc.ObjectsToSynchronizeResponse_Object_{
			Object: &agentrpc.ObjectsToSynchronizeResponse_Object{
				Source: string(path),
				Data:   data,
			},
		},
	})
}

// isHiddenDir checks if a file is in a directory, which name starts with a dot.
func isHiddenDir(filename string) bool {
	dir := path.Dir(filename)
	if dir == "." { // root directory special case
		return false
	}
	parts := strings.Split(dir, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
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
