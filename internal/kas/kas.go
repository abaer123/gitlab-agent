package kas

import (
	"context"
	"fmt"
	"io"
	"path"
	"sync/atomic"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api/apiutil"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"gitlab.com/gitlab-org/labkit/log"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"
)

const (
	// fileSizeLimit is the maximum size of:
	// - agentk's configuration files
	// - Kubernetes manifest files
	fileSizeLimit = 1024 * 1024

	agentConfigurationDirectory = "agents"
	agentConfigurationFileName  = "config.yaml"
)

type Server struct {
	// metrics must be the very first field to ensure 64-bit alignment.
	// See https://github.com/golang/go/blob/95df156e6ac53f98efd6c57e4586c1dfb43066dd/src/sync/atomic/doc.go#L46-L54
	metrics                      metrics
	Context                      context.Context
	GitalyPool                   gitaly.PoolInterface
	GitLabClient                 gitlab.ClientInterface
	AgentConfigurationPollPeriod time.Duration
	GitopsPollPeriod             time.Duration
	UsageReportingPeriod         time.Duration
}

func (s *Server) Run(ctx context.Context) {
	s.sendUsage(ctx)
}

func (s *Server) GetConfiguration(req *agentrpc.ConfigurationRequest, stream agentrpc.Kas_GetConfigurationServer) error {
	ctx := stream.Context()
	agentMeta, err := apiutil.AgentMetaFromContext(ctx)
	if err != nil {
		return err
	}
	agentInfo, err := s.GitLabClient.GetAgentInfo(ctx, agentMeta)
	if err != nil {
		return fmt.Errorf("GetAgentInfo(): %v", err)
	}
	err = wait.PollImmediateUntil(s.AgentConfigurationPollPeriod, s.sendConfiguration(agentInfo, stream), joinDone(s.Context, ctx).Done())
	if err == wait.ErrWaitTimeout {
		return nil // all good, ctx is done
	}
	return err
}

func (s *Server) sendConfiguration(agentInfo *api.AgentInfo, stream agentrpc.Kas_GetConfigurationServer) wait.ConditionFunc {
	var lastProcessedCommitId string
	p := gitaly.Poller{
		GitalyPool: s.GitalyPool,
	}
	logger := log.WithField(api.AgentId, agentInfo.ID)
	ctx := stream.Context()
	return func() (bool /*done*/, error) {
		info, err := p.Poll(ctx, &agentInfo.GitalyInfo, &agentInfo.Repository, lastProcessedCommitId, gitaly.DefaultBranch)
		if err != nil {
			logger.WithError(err).Warn("Config: repository poll failed")
			return false, nil // don't want to close the response stream, so report no error
		}
		if !info.UpdateAvailable {
			logger.WithField(api.CommitId, lastProcessedCommitId).Debug("Config: no updates")
			return false, nil
		}
		logger.WithField(api.CommitId, info.CommitID).Info("Config: new commit")
		config, err := s.fetchConfiguration(ctx, agentInfo, info.CommitID)
		if err != nil {
			logger.WithError(err).Warn("Config: failed to fetch")
			return false, nil // don't want to close the response stream, so report no error
		}
		lastProcessedCommitId = info.CommitID
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
	client, err := s.GitalyPool.CommitServiceClient(ctx, gInfo)
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
	agentMeta, err := apiutil.AgentMetaFromContext(ctx)
	if err != nil {
		return err
	}
	agentInfo, err := s.GitLabClient.GetAgentInfo(ctx, agentMeta)
	if err != nil {
		return fmt.Errorf("GetAgentInfo(): %v", err)
	}
	err = wait.PollImmediateUntil(s.GitopsPollPeriod, s.sendObjectsToSynchronize(agentInfo, stream, req.ProjectId), joinDone(s.Context, ctx).Done())
	if err == wait.ErrWaitTimeout {
		return nil // all good, ctx is done
	}
	return err
}

func (s *Server) sendObjectsToSynchronize(agentInfo *api.AgentInfo, stream agentrpc.Kas_GetObjectsToSynchronizeServer, projectId string) wait.ConditionFunc {
	var lastProcessedCommitId string
	p := gitaly.Poller{
		GitalyPool: s.GitalyPool,
	}
	logger := log.WithField(api.AgentId, agentInfo.ID)
	ctx := stream.Context()
	return func() (bool /*done*/, error) {
		// This call is made on each poll because:
		// - it checks that the agent's token is still valid
		// - repository location in Gitaly might have changed
		repoInfo, err := s.GitLabClient.GetProjectInfo(ctx, &agentInfo.Meta, projectId)
		if err != nil {
			logger.WithError(err).Warn("GitOps: failed to get project info")
			return false, nil // don't want to close the response stream, so report no error
		}
		revision := gitaly.DefaultBranch // TODO support user-specified branches/tags
		info, err := p.Poll(ctx, &repoInfo.GitalyInfo, &repoInfo.Repository, lastProcessedCommitId, revision)
		if err != nil {
			logger.WithError(err).Warn("GitOps: repository poll failed")
			return false, nil // don't want to close the response stream, so report no error
		}
		if !info.UpdateAvailable {
			logger.WithField(api.CommitId, lastProcessedCommitId).Debug("GitOps: no updates")
			return false, nil
		}
		logger.WithField(api.CommitId, info.CommitID).Info("GitOps: new commit")
		objects, err := s.fetchObjectsToSynchronize(ctx, repoInfo, info.CommitID)
		if err != nil {
			logger.WithError(err).Warn("GitOps: failed to get objects to synchronize")
			return false, nil // don't want to close the response stream, so report no error
		}
		lastProcessedCommitId = info.CommitID
		err = stream.Send(&agentrpc.ObjectsToSynchronizeResponse{
			CommitId: lastProcessedCommitId,
			Objects:  objects,
		})
		if err != nil {
			return false, err
		}
		s.metrics.IncGitopsSyncCount()
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
	if s.UsageReportingPeriod == 0 {
		return
	}
	ticker := time.NewTicker(s.UsageReportingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.sendUsageInternal(ctx); err != nil {
				log.WithError(err).Error("Failed to send usage data")
			}
		}
	}
}

func (s *Server) sendUsageInternal(ctx context.Context) error {
	m := s.metrics.Clone()
	if m.IsEmptyNotThreadSafe() {
		// No new counts
		return nil
	}
	err := s.GitLabClient.SendUsage(ctx, &gitlab.UsageData{
		GitopsSyncCount: m.gitopsSyncCount,
	})
	if err != nil {
		return err
	}
	// Subtract the increments we've just sent
	s.metrics.Subtract(m)
	return nil
}

type metrics struct {
	gitopsSyncCount int64
}

func (m *metrics) IsEmptyNotThreadSafe() bool {
	return m.gitopsSyncCount == 0
}

func (m *metrics) IncGitopsSyncCount() {
	atomic.AddInt64(&m.gitopsSyncCount, 1)
}

func (m *metrics) Clone() *metrics {
	return &metrics{
		gitopsSyncCount: atomic.LoadInt64(&m.gitopsSyncCount),
	}
}

func (m *metrics) Subtract(other *metrics) {
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
		Deployments: file.Deployments,
	}
}

// joinDone returns a context that is cancelled when c1 or c2 signal done.
// The returned context does not use c1 or c2 as a parent context to avoid inheriting any attached values,
// as this function is only meant to be used to join the done signal, not anything else. Also, it would be not
// clear which of the two to use.
//
// This helper is used here to propagate done signal from both gRPC stream's context (stream is closed/broken) and
// main program's context (program needs to stop). Polling should stop when one of this conditions happens so using
// only one of these two contexts is not good enough.
func joinDone(c1, c2 context.Context) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		select {
		case <-c1.Done():
		case <-c2.Done():
		}
	}()
	return ctx
}
