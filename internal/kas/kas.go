package kas

import (
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bmatcuk/doublestar/v2"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api/apiutil"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/grpctools"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/metric"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"
)

const (
	agentConfigurationDirectory    = ".gitlab/agents"
	agentConfigurationFileName     = "config.yaml"
	defaultGitOpsManifestNamespace = metav1.NamespaceDefault
	defaultGitOpsManifestPathGlob  = "**/*.{yaml,yml,json}"
)

var (
	// globPrefix captures glob prefix that does not contain any special characters, recognized by doublestar.Match.
	// See https://github.com/bmatcuk/doublestar#about and
	// https://pkg.go.dev/github.com/bmatcuk/doublestar/v2#Match for globbing rules.
	globPrefix = regexp.MustCompile(`^/?([^\\*?[\]{}]+)/(.*)$`)
)

type Config struct {
	Log                            *zap.Logger
	GitalyPool                     gitaly.PoolInterface
	GitLabClient                   gitlab.ClientInterface
	Registerer                     prometheus.Registerer
	ErrorTracker                   errortracking.Tracker
	AgentConfigurationPollPeriod   time.Duration
	GitopsPollPeriod               time.Duration
	UsageReportingPeriod           time.Duration
	MaxConfigurationFileSize       uint32
	MaxGitopsManifestFileSize      uint32
	MaxGitopsTotalManifestFileSize uint32
	MaxGitopsNumberOfPaths         uint32
	MaxGitopsNumberOfFiles         uint32
}

type Server struct {
	// usageMetrics must be the very first field to ensure 64-bit alignment.
	// See https://github.com/golang/go/blob/95df156e6ac53f98efd6c57e4586c1dfb43066dd/src/sync/atomic/doc.go#L46-L54
	usageMetrics                   usageMetrics
	log                            *zap.Logger
	gitalyPool                     gitaly.PoolInterface
	gitLabClient                   gitlab.ClientInterface
	errorTracker                   errortracking.Tracker
	agentConfigurationPollPeriod   time.Duration
	gitopsPollPeriod               time.Duration
	usageReportingPeriod           time.Duration
	maxConfigurationFileSize       int64
	maxGitopsManifestFileSize      int64
	maxGitopsTotalManifestFileSize int64
	maxGitopsNumberOfPaths         uint32
	maxGitopsNumberOfFiles         uint32
}

func NewServer(config Config) (*Server, func(), error) {
	toRegister := []prometheus.Collector{
		// TODO add actual metrics
	}
	cleanup, err := metric.Register(config.Registerer, toRegister...)
	if err != nil {
		return nil, nil, err
	}
	s := &Server{
		log:                            config.Log,
		gitalyPool:                     config.GitalyPool,
		gitLabClient:                   config.GitLabClient,
		errorTracker:                   config.ErrorTracker,
		agentConfigurationPollPeriod:   config.AgentConfigurationPollPeriod,
		gitopsPollPeriod:               config.GitopsPollPeriod,
		usageReportingPeriod:           config.UsageReportingPeriod,
		maxConfigurationFileSize:       int64(config.MaxConfigurationFileSize),
		maxGitopsManifestFileSize:      int64(config.MaxGitopsManifestFileSize),
		maxGitopsTotalManifestFileSize: int64(config.MaxGitopsTotalManifestFileSize),
		maxGitopsNumberOfPaths:         config.MaxGitopsNumberOfPaths,
		maxGitopsNumberOfFiles:         config.MaxGitopsNumberOfFiles,
	}
	return s, cleanup, nil
}

func (s *Server) Run(ctx context.Context) {
	s.sendUsage(ctx)
}

func (s *Server) GetConfiguration(req *agentrpc.ConfigurationRequest, stream agentrpc.Kas_GetConfigurationServer) error {
	err := retry.PollImmediateUntil(stream.Context(), s.agentConfigurationPollPeriod, s.sendConfiguration(req.CommitId, stream))
	if errors.Is(err, wait.ErrWaitTimeout) {
		return nil // all good, ctx is done
	}
	return err
}

func (s *Server) sendConfiguration(lastProcessedCommitId string, stream agentrpc.Kas_GetConfigurationServer) wait.ConditionFunc {
	p := gitaly.Poller{
		GitalyPool: s.gitalyPool,
	}
	ctx := stream.Context()
	agentMeta := apiutil.AgentMetaFromContext(ctx)
	return func() (bool /*done*/, error) {
		// This call is made on each poll because:
		// - it checks that the agent's token is still valid
		// - repository location in Gitaly might have changed
		agentInfo, err := s.gitLabClient.GetAgentInfo(ctx, agentMeta)
		switch {
		case err == nil:
		case errz.ContextDone(err):
			return false, status.Error(codes.Unavailable, "unavailable")
		case gitlab.IsForbidden(err):
			return false, status.Error(codes.PermissionDenied, "forbidden")
		case gitlab.IsUnauthorized(err):
			return false, status.Error(codes.Unauthenticated, "unauthenticated")
		default:
			s.log.Error("GetAgentInfo()", zap.Error(err))
			return false, nil // don't want to close the response stream, so report no error
		}
		l := s.log.With(logz.AgentId(agentInfo.Id), logz.ProjectId(agentInfo.Repository.GlProjectPath))
		info, err := p.Poll(ctx, &agentInfo.GitalyInfo, &agentInfo.Repository, lastProcessedCommitId, gitaly.DefaultBranch)
		if err != nil {
			if !grpctools.RequestCanceled(err) {
				l.Warn("Config: repository poll failed", zap.Error(err))
			}
			return false, nil // don't want to close the response stream, so report no error
		}
		if !info.UpdateAvailable {
			l.Debug("Config: no updates", logz.CommitId(lastProcessedCommitId))
			return false, nil
		}
		l.Info("Config: new commit", logz.CommitId(info.CommitId))
		config, err := s.fetchConfiguration(ctx, agentInfo, info.CommitId)
		if err != nil {
			if !grpctools.RequestCanceled(err) {
				l.Warn("Config: failed to fetch", zap.Error(err))
			}
			return false, nil // don't want to close the response stream, so report no error
		}
		lastProcessedCommitId = info.CommitId
		return false, stream.Send(&agentrpc.ConfigurationResponse{
			Configuration: config,
			CommitId:      lastProcessedCommitId,
		})
	}
}

// fetchConfiguration fetches agent's configuration from a corresponding repository.
// Assumes configuration is stored in ".gitlab/agents/<agent id>/config.yaml" file.
// fetchConfiguration returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
func (s *Server) fetchConfiguration(ctx context.Context, agentInfo *api.AgentInfo, revision string) (*agentcfg.AgentConfiguration, error) {
	client, err := s.gitalyPool.CommitServiceClient(ctx, &agentInfo.GitalyInfo)
	if err != nil {
		return nil, fmt.Errorf("CommitServiceClient: %w", err) // wrap
	}
	filename := path.Join(agentConfigurationDirectory, agentInfo.Name, agentConfigurationFileName)
	f := gitaly.PathFetcher{
		Client: client,
	}
	configYAML, err := f.FetchSingleFile(ctx, &agentInfo.Repository, []byte(revision), []byte(filename), s.maxConfigurationFileSize)
	if err != nil {
		return nil, fmt.Errorf("fetch agent configuration: %w", err) // wrap
	}
	if configYAML == nil {
		return nil, fmt.Errorf("configuration file not found: %q", filename)
	}
	configFile, err := parseYAMLToConfiguration(configYAML)
	if err != nil {
		return nil, fmt.Errorf("parse agent configuration: %v", err)
	}
	err = configFile.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid agent configuration: %v", err)
	}
	agentConfig := defaultAndExtractAgentConfiguration(configFile)
	return agentConfig, nil
}

func (s *Server) GetObjectsToSynchronize(req *agentrpc.ObjectsToSynchronizeRequest, stream agentrpc.Kas_GetObjectsToSynchronizeServer) error {
	ctx := stream.Context()
	agentMeta := apiutil.AgentMetaFromContext(ctx)
	agentInfo, err := s.gitLabClient.GetAgentInfo(ctx, agentMeta)
	switch {
	case err == nil:
	case errz.ContextDone(err):
		return status.Error(codes.Unavailable, "unavailable")
	case gitlab.IsForbidden(err):
		return status.Error(codes.PermissionDenied, "forbidden")
	case gitlab.IsUnauthorized(err):
		return status.Error(codes.Unauthenticated, "unauthenticated")
	default:
		s.log.Error("GetAgentInfo()", zap.Error(err))
		return status.Error(codes.Unavailable, "unavailable")
	}
	numberOfPaths := uint32(len(req.Paths))
	if numberOfPaths > s.maxGitopsNumberOfPaths {
		return status.Errorf(codes.InvalidArgument, "maximum number of GitOps paths per manifest project is %d, but %d was requested", s.maxGitopsNumberOfPaths, numberOfPaths)
	}
	err = retry.PollImmediateUntil(ctx, s.gitopsPollPeriod, s.sendObjectsToSynchronize(agentInfo, req, stream))
	if errors.Is(err, wait.ErrWaitTimeout) {
		return nil // all good, ctx is done
	}
	return err
}

func (s *Server) sendObjectsToSynchronize(agentInfo *api.AgentInfo, req *agentrpc.ObjectsToSynchronizeRequest, stream agentrpc.Kas_GetObjectsToSynchronizeServer) wait.ConditionFunc {
	p := gitaly.Poller{
		GitalyPool: s.gitalyPool,
	}
	ctx := stream.Context()
	projectId := req.ProjectId
	lastProcessedCommitId := req.CommitId
	l := s.log.With(logz.AgentId(agentInfo.Id), logz.ProjectId(projectId))
	return func() (bool /*done*/, error) {
		// This call is made on each poll because:
		// - it checks that the agent's token is still valid
		// - repository location in Gitaly might have changed
		repoInfo, err := s.gitLabClient.GetProjectInfo(ctx, &agentInfo.Meta, projectId)
		switch {
		case err == nil:
		case errz.ContextDone(err):
			return false, status.Error(codes.Unavailable, "unavailable")
		case gitlab.IsForbidden(err):
			return false, status.Error(codes.PermissionDenied, "forbidden")
		case gitlab.IsUnauthorized(err):
			return false, status.Error(codes.Unauthenticated, "unauthenticated")
		default:
			l.Warn("GitOps: failed to get project info", zap.Error(err))
			return false, nil // don't want to close the response stream, so report no error
		}
		revision := gitaly.DefaultBranch // TODO support user-specified branches/tags
		info, err := p.Poll(ctx, &repoInfo.GitalyInfo, &repoInfo.Repository, lastProcessedCommitId, revision)
		if err != nil {
			if !grpctools.RequestCanceled(err) {
				l.Warn("GitOps: repository poll failed", zap.Error(err))
			}
			return false, nil // don't want to close the response stream, so report no error
		}
		if !info.UpdateAvailable {
			l.Debug("GitOps: no updates", logz.CommitId(lastProcessedCommitId))
			return false, nil
		}
		// Create a new l variable, don't want to mutate the one from the outer scope
		l := l.With(logz.CommitId(info.CommitId)) // nolint:govet
		l.Info("GitOps: new commit")
		objects, err := s.fetchObjectsToSynchronize(ctx, repoInfo, req.Paths, info.CommitId)
		if err != nil {
			if !grpctools.RequestCanceled(err) {
				l.Warn("GitOps: failed to get objects to synchronize", zap.Error(err))
			}
			return false, nil // don't want to close the response stream, so report no error
		}
		l.Info("GitOps: fetched files", logz.NumberOfFiles(len(objects)))
		lastProcessedCommitId = info.CommitId
		err = stream.Send(&agentrpc.ObjectsToSynchronizeResponse{
			CommitId: lastProcessedCommitId,
			Objects:  objects,
		})
		if err != nil {
			return false, err
		}
		s.usageMetrics.IncGitopsSyncCount()
		return false, nil
	}
}

// fetchObjectsToSynchronize returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
func (s *Server) fetchObjectsToSynchronize(ctx context.Context, repoInfo *api.ProjectInfo, paths []*agentcfg.PathCF, revision string) ([]*agentrpc.ObjectToSynchronize, error) {
	client, err := s.gitalyPool.CommitServiceClient(ctx, &repoInfo.GitalyInfo)
	if err != nil {
		return nil, fmt.Errorf("CommitServiceClient: %w", err) // wrap
	}
	f := gitaly.PathFetcher{
		Client: client,
	}
	v := &objectsToSynchronizeVisitor{
		remainingTotalFileSize: s.maxGitopsTotalManifestFileSize,
		fileSizeLimit:          s.maxGitopsManifestFileSize,
		maxNumberOfFiles:       s.maxGitopsNumberOfFiles,
	}
	for _, p := range paths {
		repoPath, recursive, glob := globToGitaly(p.Glob)
		v.glob = glob // set new glob for each path
		err := f.Visit(ctx, &repoInfo.Repository, []byte(revision), repoPath, recursive, v)
		if err != nil {
			return nil, fmt.Errorf("fetch: %w", err) // wrap
		}
	}
	return v.objects, nil
}

func (s *Server) sendUsage(ctx context.Context) {
	if s.usageReportingPeriod == 0 {
		return
	}
	ticker := time.NewTicker(s.usageReportingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.sendUsageInternal(ctx); err != nil {
				if !errz.ContextDone(err) {
					s.log.Warn("Failed to send usage data", zap.Error(err))
					s.errorTracker.Capture(err, errortracking.WithContext(ctx))
				}
			}
		}
	}
}

func (s *Server) sendUsageInternal(ctx context.Context) error {
	m := s.usageMetrics.Clone()
	if m.IsEmptyNotThreadSafe() {
		// No new counts
		return nil
	}
	err := s.gitLabClient.SendUsage(ctx, &gitlab.UsageData{
		GitopsSyncCount: m.gitopsSyncCount,
	})
	if err != nil {
		return err // don't wrap
	}
	// Subtract the increments we've just sent
	s.usageMetrics.Subtract(m)
	return nil
}

type objectsToSynchronizeVisitor struct {
	glob                   string
	remainingTotalFileSize int64
	fileSizeLimit          int64
	maxNumberOfFiles       uint32
	numberOfFiles          uint32
	objects                []*agentrpc.ObjectToSynchronize
}

func (v *objectsToSynchronizeVisitor) VisitEntry(entry *gitalypb.TreeEntry) (bool /* download? */, int64 /* max size */, error) {
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

func (v *objectsToSynchronizeVisitor) VisitBlob(blob gitaly.Blob) (bool /* done? */, error) {
	v.remainingTotalFileSize -= int64(len(blob.Data))
	if v.remainingTotalFileSize < 0 {
		// This should never happen because we told Gitaly the maximum file size that we'd like to get.
		// i.e. we should have gotten an error from Gitaly if file is bigger than the limit.
		return false, errors.New("unexpected negative remaining total file size")
	}
	v.objects = append(v.objects, &agentrpc.ObjectToSynchronize{
		Object: blob.Data,
		Source: string(blob.Path),
	})
	return false, nil
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

type usageMetrics struct {
	gitopsSyncCount int64
}

func (m *usageMetrics) IsEmptyNotThreadSafe() bool {
	return m.gitopsSyncCount == 0
}

func (m *usageMetrics) IncGitopsSyncCount() {
	atomic.AddInt64(&m.gitopsSyncCount, 1)
}

func (m *usageMetrics) Clone() *usageMetrics {
	return &usageMetrics{
		gitopsSyncCount: atomic.LoadInt64(&m.gitopsSyncCount),
	}
}

func (m *usageMetrics) Subtract(other *usageMetrics) {
	atomic.AddInt64(&m.gitopsSyncCount, -other.gitopsSyncCount)
}

func parseYAMLToConfiguration(configYAML []byte) (*agentcfg.ConfigurationFile, error) {
	configJSON, err := yaml.YAMLToJSON(configYAML)
	if err != nil {
		return nil, fmt.Errorf("YAMLToJSON: %v", err)
	}
	configFile := &agentcfg.ConfigurationFile{}
	err = protojson.Unmarshal(configJSON, configFile)
	if err != nil {
		return nil, fmt.Errorf("protojson.Unmarshal: %v", err)
	}
	return configFile, nil
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

func defaultAndExtractAgentConfiguration(file *agentcfg.ConfigurationFile) *agentcfg.AgentConfiguration {
	protodefault.NotNil(&file.Gitops)
	for _, project := range file.Gitops.ManifestProjects {
		applyDefaultsToManifestProject(project)
	}
	return &agentcfg.AgentConfiguration{
		Gitops:        file.Gitops,
		Observability: file.Observability,
	}
}

func applyDefaultsToManifestProject(project *agentcfg.ManifestProjectCF) {
	protodefault.String(&project.DefaultNamespace, defaultGitOpsManifestNamespace)
	if len(project.Paths) == 0 {
		project.Paths = []*agentcfg.PathCF{
			{
				Glob: defaultGitOpsManifestPathGlob,
			},
		}
	}
}
