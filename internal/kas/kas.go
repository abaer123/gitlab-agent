package kas

import (
	"context"
	"fmt"
	"io"
	"path"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api/apiutil"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/metric"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/sentryapi"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"
)

const (
	// fileSizeLimit is the maximum size of:
	// - agentk's configuration files
	// - Kubernetes manifest files
	fileSizeLimit = 1024 * 1024

	agentConfigurationDirectory = ".gitlab/agents"
	agentConfigurationFileName  = "config.yaml"
)

type Config struct {
	Log                          *zap.Logger
	GitalyPool                   gitaly.PoolInterface
	GitLabClient                 gitlab.ClientInterface
	AgentConfigurationPollPeriod time.Duration
	GitopsPollPeriod             time.Duration
	UsageReportingPeriod         time.Duration
	Registerer                   prometheus.Registerer
	Sentry                       sentryapi.Hub
}

type Server struct {
	// usageMetrics must be the very first field to ensure 64-bit alignment.
	// See https://github.com/golang/go/blob/95df156e6ac53f98efd6c57e4586c1dfb43066dd/src/sync/atomic/doc.go#L46-L54
	usageMetrics                 usageMetrics
	log                          *zap.Logger
	gitalyPool                   gitaly.PoolInterface
	gitLabClient                 gitlab.ClientInterface
	agentConfigurationPollPeriod time.Duration
	gitopsPollPeriod             time.Duration
	usageReportingPeriod         time.Duration
	sentry                       sentryapi.Hub
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
		log:                          config.Log,
		gitalyPool:                   config.GitalyPool,
		gitLabClient:                 config.GitLabClient,
		agentConfigurationPollPeriod: config.AgentConfigurationPollPeriod,
		gitopsPollPeriod:             config.GitopsPollPeriod,
		usageReportingPeriod:         config.UsageReportingPeriod,
		sentry:                       config.Sentry,
	}
	return s, cleanup, nil
}

func (s *Server) Run(ctx context.Context) {
	s.sendUsage(ctx)
}

func (s *Server) GetConfiguration(req *agentrpc.ConfigurationRequest, stream agentrpc.Kas_GetConfigurationServer) error {
	err := wait.PollImmediateUntil(s.agentConfigurationPollPeriod, s.sendConfiguration(req.CommitId, stream), stream.Context().Done())
	if err == wait.ErrWaitTimeout {
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
		case gitlab.IsForbidden(err):
			return false, status.Error(codes.PermissionDenied, "forbidden")
		case gitlab.IsUnauthorized(err):
			return false, status.Error(codes.Unauthenticated, "unauthenticated")
		default:
			s.log.Error("GetAgentInfo()", zap.Error(err))
			return false, nil // don't want to close the response stream, so report no error
		}
		l := s.log.With(logz.AgentId(agentInfo.Id), logz.ProjectPath(agentInfo.Repository.GlProjectPath))
		info, err := p.Poll(ctx, &agentInfo.GitalyInfo, &agentInfo.Repository, lastProcessedCommitId, gitaly.DefaultBranch)
		if err != nil {
			l.Warn("Config: repository poll failed", zap.Error(err))
			return false, nil // don't want to close the response stream, so report no error
		}
		if !info.UpdateAvailable {
			l.Debug("Config: no updates", logz.CommitId(lastProcessedCommitId))
			return false, nil
		}
		l.Info("Config: new commit", logz.CommitId(info.CommitId))
		config, err := s.fetchConfiguration(ctx, agentInfo, info.CommitId)
		if err != nil {
			l.Warn("Config: failed to fetch", zap.Error(err))
			return false, nil // don't want to close the response stream, so report no error
		}
		lastProcessedCommitId = info.CommitId
		return false, stream.Send(config)
	}
}

// fetchConfiguration fetches agent's configuration from a corresponding repository.
// Assumes configuration is stored in "agents/<agent id>/config.yaml" file.
func (s *Server) fetchConfiguration(ctx context.Context, agentInfo *api.AgentInfo, revision string) (*agentrpc.ConfigurationResponse, error) {
	filename := path.Join(agentConfigurationDirectory, agentInfo.Name, agentConfigurationFileName)
	configYAML, err := s.fetchSingleFile(ctx, &agentInfo.GitalyInfo, &agentInfo.Repository, filename, revision)
	if err != nil {
		return nil, fmt.Errorf("fetch agent configuration: %v", err)
	}
	if configYAML == nil {
		return nil, fmt.Errorf("configuration file not found: %q", filename)
	}
	configFile, err := parseYAMLToConfiguration(configYAML)
	if err != nil {
		return nil, fmt.Errorf("parse agent configuration: %v", err)
	}
	agentConfig := extractAgentConfiguration(configFile)
	return &agentrpc.ConfigurationResponse{
		Configuration: agentConfig,
	}, nil
}

// fetchSingleFile fetches the latest revision of a single file.
// Returned data slice is nil if file was not found and is empty if the file is empty.
func (s *Server) fetchSingleFile(ctx context.Context, gInfo *api.GitalyInfo, repo *gitalypb.Repository, filename, revision string) ([]byte, error) {
	client, err := s.gitalyPool.CommitServiceClient(ctx, gInfo)
	if err != nil {
		return nil, fmt.Errorf("CommitServiceClient: %v", err)
	}
	treeEntryReq := &gitalypb.TreeEntryRequest{
		Repository: repo,
		Revision:   []byte(revision),
		Path:       []byte(filename),
		Limit:      fileSizeLimit,
	}
	teResp, err := client.TreeEntry(ctx, treeEntryReq)
	if err != nil {
		return nil, fmt.Errorf("TreeEntry: %v", err)
	}
	var fileData []byte
	for {
		entry, err := teResp.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("TreeEntry.Recv: %v", err)
		}
		fileData = append(fileData, entry.Data...)
	}
	return fileData, nil
}

func (s *Server) GetObjectsToSynchronize(req *agentrpc.ObjectsToSynchronizeRequest, stream agentrpc.Kas_GetObjectsToSynchronizeServer) error {
	ctx := stream.Context()
	agentMeta := apiutil.AgentMetaFromContext(ctx)
	agentInfo, err := s.gitLabClient.GetAgentInfo(ctx, agentMeta)
	switch {
	case err == nil:
	case gitlab.IsForbidden(err):
		return status.Error(codes.PermissionDenied, "forbidden")
	case gitlab.IsUnauthorized(err):
		return status.Error(codes.Unauthenticated, "unauthenticated")
	default:
		s.log.Error("GetAgentInfo()", zap.Error(err))
		return status.Error(codes.Unavailable, "unavailable")
	}
	err = wait.PollImmediateUntil(s.gitopsPollPeriod, s.sendObjectsToSynchronize(agentInfo, stream, req.ProjectId, req.CommitId), ctx.Done())
	if err == wait.ErrWaitTimeout {
		return nil // all good, ctx is done
	}
	return err
}

func (s *Server) sendObjectsToSynchronize(agentInfo *api.AgentInfo, stream agentrpc.Kas_GetObjectsToSynchronizeServer, projectId, lastProcessedCommitId string) wait.ConditionFunc {
	p := gitaly.Poller{
		GitalyPool: s.gitalyPool,
	}
	ctx := stream.Context()
	l := s.log.With(logz.AgentId(agentInfo.Id), logz.ProjectId(projectId))
	return func() (bool /*done*/, error) {
		// This call is made on each poll because:
		// - it checks that the agent's token is still valid
		// - repository location in Gitaly might have changed
		repoInfo, err := s.gitLabClient.GetProjectInfo(ctx, &agentInfo.Meta, projectId)
		switch {
		case err == nil:
		case gitlab.IsForbidden(err):
			return false, status.Error(codes.PermissionDenied, "forbidden")
		case gitlab.IsUnauthorized(err):
			return false, status.Error(codes.Unauthenticated, "unauthenticated")
		default:
			l.Warn("GitOps: failed to get project info", zap.Error(err))
			return false, nil // don't want to close the response stream, so report no error
		}
		l = l.With(logz.ProjectPath(repoInfo.Repository.GlRepository))
		revision := gitaly.DefaultBranch // TODO support user-specified branches/tags
		info, err := p.Poll(ctx, &repoInfo.GitalyInfo, &repoInfo.Repository, lastProcessedCommitId, revision)
		if err != nil {
			l.Warn("GitOps: repository poll failed", zap.Error(err))
			return false, nil // don't want to close the response stream, so report no error
		}
		if !info.UpdateAvailable {
			l.Debug("GitOps: no updates", logz.CommitId(lastProcessedCommitId))
			return false, nil
		}
		l.Info("GitOps: new commit", logz.CommitId(info.CommitId))
		objects, err := s.fetchObjectsToSynchronize(ctx, repoInfo, info.CommitId)
		if err != nil {
			l.Warn("GitOps: failed to get objects to synchronize", zap.Error(err))
			return false, nil // don't want to close the response stream, so report no error
		}
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

func (s *Server) fetchObjectsToSynchronize(ctx context.Context, repoInfo *api.ProjectInfo, revision string) ([]*agentrpc.ObjectToSynchronize, error) {
	// TODO fetching just one file with a hardcoded name is a shortcut to cut scope
	filename := "manifest.yaml"
	manifestYAML, err := s.fetchSingleFile(ctx, &repoInfo.GitalyInfo, &repoInfo.Repository, filename, revision)
	if err != nil {
		return nil, err
	}
	if manifestYAML == nil {
		return nil, fmt.Errorf("manifest file not found: %q", filename)
	}
	return []*agentrpc.ObjectToSynchronize{
		{
			Object: manifestYAML,
			Source: filename,
		},
	}, nil
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
				s.log.Warn("Failed to send usage data", zap.Error(err))
				s.sentry.CaptureException(err)
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
		return err
	}
	// Subtract the increments we've just sent
	s.usageMetrics.Subtract(m)
	return nil
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

func extractAgentConfiguration(file *agentcfg.ConfigurationFile) *agentcfg.AgentConfiguration {
	return &agentcfg.AgentConfiguration{
		Gitops:        file.Gitops,
		Observability: file.Observability,
	}
}
