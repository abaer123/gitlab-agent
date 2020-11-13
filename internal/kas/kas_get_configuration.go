package kas

import (
	"context"
	"fmt"
	"path"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api/apiutil"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"
)

const (
	agentConfigurationDirectory = ".gitlab/agents"
	agentConfigurationFileName  = "config.yaml"

	defaultGitOpsManifestNamespace = metav1.NamespaceDefault
	defaultGitOpsManifestPathGlob  = "**/*.{yaml,yml,json}"
)

func (s *Server) GetConfiguration(req *agentrpc.ConfigurationRequest, stream agentrpc.Kas_GetConfigurationServer) error {
	return s.pollImmediateUntil(stream.Context(), s.agentConfigurationPollPeriod, s.sendConfiguration(req.CommitId, stream))
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
			if !grpctool.RequestCanceled(err) {
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
			if !grpctool.RequestCanceled(err) {
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
	configYAML, err := f.FetchFile(ctx, &agentInfo.Repository, []byte(revision), []byte(filename), s.maxConfigurationFileSize)
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
