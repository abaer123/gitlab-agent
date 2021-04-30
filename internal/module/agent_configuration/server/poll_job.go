package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_configuration/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/agent_tracker"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/errz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/retry"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/yaml"
)

const (
	agentConfigurationDirectory = ".gitlab/agents"
	agentConfigurationFileName  = "config.yaml"
)

type pollJob struct {
	ctx                      context.Context
	log                      *zap.Logger
	api                      modserver.API
	gitaly                   gitaly.PoolInterface
	agentRegisterer          agent_tracker.Registerer
	server                   rpc.AgentConfiguration_GetConfigurationServer
	agentToken               api.AgentToken
	maxConfigurationFileSize int64
	lastProcessedCommitId    string
	connectedAgentInfo       *agent_tracker.ConnectedAgentInfo
	connectionRegistered     bool
}

func (j *pollJob) Attempt() (error, retry.AttemptResult) {
	// This call is made on each poll because:
	// - it checks that the agent's token is still valid
	// - repository location in Gitaly might have changed
	agentInfo, err := j.api.GetAgentInfo(j.ctx, j.log, j.agentToken)
	if err != nil {
		return err, retry.Done
	}
	if !j.connectionRegistered { // only register once
		j.connectedAgentInfo.AgentId = agentInfo.Id
		j.connectedAgentInfo.ProjectId = agentInfo.ProjectId
		j.agentRegisterer.RegisterConnection(j.ctx, j.connectedAgentInfo)
		j.connectionRegistered = true
	}
	log := j.log.With(logz.AgentId(agentInfo.Id), logz.ProjectId(agentInfo.Repository.GlProjectPath)) // nolint:govet
	p, err := j.gitaly.Poller(j.ctx, &agentInfo.GitalyInfo)
	if err != nil {
		j.api.HandleProcessingError(j.ctx, log, "Config: Poller", err)
		return nil, retry.Backoff
	}
	info, err := p.Poll(j.ctx, &agentInfo.Repository, j.lastProcessedCommitId, gitaly.DefaultBranch)
	if err != nil {
		j.api.HandleProcessingError(j.ctx, log, "Config: repository poll failed", err)
		return nil, retry.Backoff
	}
	if !info.UpdateAvailable {
		log.Debug("Config: no updates", logz.CommitId(j.lastProcessedCommitId))
		return nil, retry.Continue
	}
	log.Info("Config: new commit", logz.CommitId(info.CommitId))
	config, err := j.fetchConfiguration(j.ctx, agentInfo, info.CommitId)
	if err != nil {
		j.api.HandleProcessingError(j.ctx, log, "Config: failed to fetch", err)
		var ue *errz.UserError
		if errors.As(err, &ue) {
			// return the error to the client because it's a user error
			return status.Errorf(codes.FailedPrecondition, "Config: %v", err), retry.Done
		}
		return nil, retry.Backoff
	}
	err = j.server.Send(&rpc.ConfigurationResponse{
		Configuration: config,
		CommitId:      info.CommitId,
	})
	if err != nil {
		return j.api.HandleSendError(log, "Config: failed to send config", err), retry.Done
	}
	j.lastProcessedCommitId = info.CommitId
	return nil, retry.Continue
}

func (j *pollJob) Cleanup() {
	if !j.connectionRegistered {
		return
	}
	j.agentRegisterer.UnregisterConnection(context.Background(), j.connectedAgentInfo)
}

// fetchConfiguration fetches agent's configuration from a corresponding repository.
// Assumes configuration is stored in ".gitlab/agents/<agent id>/config.yaml" file.
// fetchConfiguration returns a wrapped context.Canceled, context.DeadlineExceeded or gRPC error if ctx signals done and interrupts a running gRPC call.
func (j *pollJob) fetchConfiguration(ctx context.Context, agentInfo *api.AgentInfo, revision string) (*agentcfg.AgentConfiguration, error) {
	pf, err := j.gitaly.PathFetcher(ctx, &agentInfo.GitalyInfo)
	if err != nil {
		return nil, fmt.Errorf("PathFetcher: %w", err) // wrap
	}
	filename := path.Join(agentConfigurationDirectory, agentInfo.Name, agentConfigurationFileName)
	configYAML, err := pf.FetchFile(ctx, &agentInfo.Repository, []byte(revision), []byte(filename), j.maxConfigurationFileSize)
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
	err = configFile.Validate(true)
	if err != nil {
		return nil, errz.NewUserErrorWithCause(err, "invalid agent configuration")
	}
	return &agentcfg.AgentConfiguration{
		Gitops:        configFile.Gitops,
		Observability: configFile.Observability,
		Cilium:        configFile.Cilium,
		AgentId:       agentInfo.Id,
		ProjectId:     agentInfo.ProjectId,
	}, nil
}

func parseYAMLToConfiguration(configYAML []byte) (*agentcfg.ConfigurationFile, error) {
	configJSON, err := yaml.YAMLToJSON(configYAML)
	if err != nil {
		return nil, fmt.Errorf("YAMLToJSON: %v", err)
	}
	configFile := &agentcfg.ConfigurationFile{}
	if bytes.Equal(configJSON, []byte("null")) {
		// Empty config
		return configFile, nil
	}
	err = protojson.Unmarshal(configJSON, configFile)
	if err != nil {
		return nil, fmt.Errorf("protojson.Unmarshal: %v", err)
	}
	return configFile, nil
}
