package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
	gapi "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_configuration"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/mathz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"
)

type server struct {
	rpc.UnimplementedAgentConfigurationServer
	api                        modserver.API
	gitaly                     gitaly.PoolInterface
	gitLabClient               gitlab.ClientInterface
	agentRegisterer            agent_tracker.Registerer
	maxConfigurationFileSize   int64
	getConfigurationPollConfig retry.PollConfigFactory
}

func (s *server) GetConfiguration(req *rpc.ConfigurationRequest, server rpc.AgentConfiguration_GetConfigurationServer) error {
	connectedAgentInfo := &agent_tracker.ConnectedAgentInfo{
		AgentMeta:    req.AgentMeta,
		ConnectedAt:  timestamppb.Now(),
		ConnectionId: mathz.Int63(),
	}
	defer s.maybeUnregisterAgent(connectedAgentInfo)
	ctx := server.Context()
	log := grpctool.LoggerFromContext(ctx)
	agentToken := api.AgentTokenFromContext(ctx)
	lastProcessedCommitId := req.CommitId
	return s.api.PollWithBackoff(server, s.getConfigurationPollConfig(), func() (error, retry.AttemptResult) {
		// This call is made on each poll because:
		// - it checks that the agent's token is still valid
		// - repository location in Gitaly might have changed
		agentInfo, err := s.api.GetAgentInfo(ctx, log, agentToken)
		if err != nil {
			return err, retry.Done
		}
		s.maybeRegisterAgent(ctx, connectedAgentInfo, agentInfo)
		// re-define log to avoid accidentally using the old one
		log := log.With(logz.AgentId(agentInfo.Id), logz.ProjectId(agentInfo.Repository.GlProjectPath)) // nolint:govet
		info, err := s.poll(ctx, agentInfo, lastProcessedCommitId)
		if err != nil {
			s.api.HandleProcessingError(ctx, log, "Config: repository poll failed", err)
			return nil, retry.Backoff
		}
		if !info.UpdateAvailable {
			log.Debug("Config: no updates", logz.CommitId(lastProcessedCommitId))
			return nil, retry.Continue
		}
		log.Info("Config: new commit", logz.CommitId(info.CommitId))
		configFile, err := s.fetchConfiguration(ctx, agentInfo, info.CommitId)
		if err != nil {
			s.api.HandleProcessingError(ctx, log, "Config: failed to fetch", err)
			var ue errz.UserError
			if errors.As(err, &ue) {
				// return the error to the client because it's a user error
				return status.Errorf(codes.FailedPrecondition, "Config: %v", err), retry.Done
			}
			return nil, retry.Backoff
		}
		var wg wait.Group
		defer wg.Wait()
		wg.Start(func() {
			err := gapi.PostAgentConfiguration(ctx, s.gitLabClient, agentInfo.Id, configFile) // nolint:govet
			if err != nil {
				s.api.HandleProcessingError(ctx, log, "Failed to notify GitLab of new agent configuration", err)
			}
		})
		err = s.sendConfigResponse(server, agentInfo, configFile, info.CommitId)
		if err != nil {
			return s.api.HandleSendError(log, "Config: failed to send config", err), retry.Done
		}
		lastProcessedCommitId = info.CommitId
		return nil, retry.Continue
	})
}

func (s *server) poll(ctx context.Context, agentInfo *api.AgentInfo, lastProcessedCommitId string) (*gitaly.PollInfo, error) {
	p, err := s.gitaly.Poller(ctx, &agentInfo.GitalyInfo)
	if err != nil {
		return nil, err
	}
	return p.Poll(ctx, agentInfo.Repository, lastProcessedCommitId, gitaly.DefaultBranch)
}

func (s *server) sendConfigResponse(server rpc.AgentConfiguration_GetConfigurationServer,
	agentInfo *api.AgentInfo, configFile *agentcfg.ConfigurationFile, commitId string) error {
	return server.Send(&rpc.ConfigurationResponse{
		Configuration: &agentcfg.AgentConfiguration{
			Gitops:        configFile.Gitops,
			Observability: configFile.Observability,
			Cilium:        configFile.Cilium,
			AgentId:       agentInfo.Id,
			ProjectId:     agentInfo.ProjectId,
		},
		CommitId: commitId,
	})
}

// fetchConfiguration fetches agent's configuration from a corresponding repository.
// Assumes configuration is stored in ".gitlab/agents/<agent id>/config.yaml" file.
// fetchConfiguration returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
func (s *server) fetchConfiguration(ctx context.Context, agentInfo *api.AgentInfo, commitId string) (*agentcfg.ConfigurationFile, error) {
	pf, err := s.gitaly.PathFetcher(ctx, &agentInfo.GitalyInfo)
	if err != nil {
		return nil, fmt.Errorf("PathFetcher: %w", err) // wrap
	}
	filename := path.Join(agent_configuration.Directory, agentInfo.Name, agent_configuration.FileName)
	configYAML, err := pf.FetchFile(ctx, agentInfo.Repository, []byte(commitId), []byte(filename), s.maxConfigurationFileSize)
	if err != nil {
		switch gitaly.ErrorCodeFromError(err) { // nolint:exhaustive
		case gitaly.NotFound, gitaly.FileTooBig, gitaly.UnexpectedTreeEntryType:
			return nil, errz.NewUserErrorWithCause(err, "agent configuration file")
		default:
			return nil, fmt.Errorf("fetch agent configuration: %w", err) // wrap
		}
	}
	configFile, err := parseYAMLToConfiguration(configYAML)
	if err != nil {
		return nil, errz.NewUserErrorWithCause(err, "failed to parse agent configuration")
	}
	err = configFile.Validate()
	if err != nil {
		return nil, errz.NewUserErrorWithCause(err, "invalid agent configuration")
	}
	return configFile, nil
}

func (s *server) maybeRegisterAgent(ctx context.Context, connectedAgentInfo *agent_tracker.ConnectedAgentInfo, agentInfo *api.AgentInfo) {
	if connectedAgentInfo.AgentId != 0 {
		return
	}
	connectedAgentInfo.AgentId = agentInfo.Id
	connectedAgentInfo.ProjectId = agentInfo.ProjectId
	s.agentRegisterer.RegisterConnection(ctx, connectedAgentInfo)
}

func (s *server) maybeUnregisterAgent(connectedAgentInfo *agent_tracker.ConnectedAgentInfo) {
	if connectedAgentInfo.AgentId == 0 {
		return
	}
	s.agentRegisterer.UnregisterConnection(context.Background(), connectedAgentInfo)
}

func parseYAMLToConfiguration(configYAML []byte) (*agentcfg.ConfigurationFile, error) {
	configJSON, err := yaml.YAMLToJSON(configYAML)
	if err != nil {
		return nil, fmt.Errorf("YAMLToJSON: %w", err)
	}
	configFile := &agentcfg.ConfigurationFile{}
	if bytes.Equal(configJSON, []byte("null")) {
		// Empty config
		return configFile, nil
	}
	err = protojson.Unmarshal(configJSON, configFile)
	if err != nil {
		return nil, fmt.Errorf("protojson.Unmarshal: %w", err)
	}
	return configFile, nil
}
