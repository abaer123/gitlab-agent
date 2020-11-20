package kas

import (
	"context"
	"fmt"
	"path"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api/apiutil"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/protodefault"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
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
	ctx := stream.Context()
	agentMeta := apiutil.AgentMetaFromContext(ctx)
	return func() (bool /*done*/, error) {
		// This call is made on each poll because:
		// - it checks that the agent's token is still valid
		// - repository location in Gitaly might have changed
		agentInfo, err, retErr := s.getAgentInfo(ctx, agentMeta, true) // don't want to close the response stream, so report no error
		if retErr {
			return false, err
		}
		l := s.log.With(logz.AgentId(agentInfo.Id), logz.ProjectId(agentInfo.Repository.GlProjectPath))
		p, err := s.gitalyPool.Poller(ctx, &agentInfo.GitalyInfo)
		if err != nil {
			s.handleProcessingError(ctx, l, "Config: Poller", err)
			return false, nil // don't want to close the response stream, so report no error
		}
		info, err := p.Poll(ctx, &agentInfo.Repository, lastProcessedCommitId, gitaly.DefaultBranch)
		if err != nil {
			s.handleProcessingError(ctx, l, "Config: repository poll failed", err)
			return false, nil // don't want to close the response stream, so report no error
		}
		if !info.UpdateAvailable {
			l.Debug("Config: no updates", logz.CommitId(lastProcessedCommitId))
			return false, nil // don't want to close the response stream, so report no error
		}
		l.Info("Config: new commit", logz.CommitId(info.CommitId))
		config, err := s.fetchConfiguration(ctx, agentInfo, info.CommitId)
		if err != nil {
			s.handleProcessingError(ctx, l, "Config: failed to fetch", err)
			return false, nil // don't want to close the response stream, so report no error
		}
		err = stream.Send(&agentrpc.ConfigurationResponse{
			Configuration: config,
			CommitId:      info.CommitId,
		})
		if err != nil {
			return false, s.handleFailedSend(l, "Config: failed to send config", err)
		}
		lastProcessedCommitId = info.CommitId
		return false, nil
	}
}

// fetchConfiguration fetches agent's configuration from a corresponding repository.
// Assumes configuration is stored in ".gitlab/agents/<agent id>/config.yaml" file.
// fetchConfiguration returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
func (s *Server) fetchConfiguration(ctx context.Context, agentInfo *api.AgentInfo, revision string) (*agentcfg.AgentConfiguration, error) {
	pf, err := s.gitalyPool.PathFetcher(ctx, &agentInfo.GitalyInfo)
	if err != nil {
		return nil, fmt.Errorf("PathFetcher: %w", err) // wrap
	}
	filename := path.Join(agentConfigurationDirectory, agentInfo.Name, agentConfigurationFileName)
	configYAML, err := pf.FetchFile(ctx, &agentInfo.Repository, []byte(revision), []byte(filename), s.maxConfigurationFileSize)
	if err != nil {
		return nil, fmt.Errorf("fetch agent configuration: %w", err) // wrap
	}
	if configYAML == nil {
		return nil, errz.NewUserErrorf("configuration file not found: %s", filename)
	}
	configFile, err := parseYAMLToConfiguration(configYAML)
	if err != nil {
		return nil, errz.NewUserErrorWithCause(err, "failed to parse agent configuration")
	}
	err = configFile.Validate()
	if err != nil {
		return nil, errz.NewUserErrorWithCause(err, "invalid agent configuration")
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
