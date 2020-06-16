package kgb

import (
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/api/apiutil"
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

type GitLabClient interface {
	GetAgentInfo(context.Context, *api.AgentMeta) (*api.AgentInfo, error)
	GetProjectInfo(ctx context.Context, agentMeta *api.AgentMeta, projectId string) (*api.ProjectInfo, error)
}

type Agent struct {
	ReloadConfigurationPeriod time.Duration
	CommitServiceClient       gitalypb.CommitServiceClient
	GitLabClient              GitLabClient
}

func (a *Agent) GetConfiguration(req *agentrpc.ConfigurationRequest, stream agentrpc.GitLabService_GetConfigurationServer) error {
	ctx := stream.Context()
	agentMeta, err := apiutil.AgentMetaFromContext(ctx)
	if err != nil {
		return err
	}
	agentInfo, err := a.GitLabClient.GetAgentInfo(ctx, agentMeta)
	if err != nil {
		return err
	}
	err = wait.PollImmediateUntil(a.ReloadConfigurationPeriod, a.sendConfiguration(agentInfo, stream), ctx.Done())
	if err == wait.ErrWaitTimeout {
		return nil // all good, ctx is done
	}
	return err
}

func (a *Agent) sendConfiguration(agentInfo *api.AgentInfo, stream agentrpc.GitLabService_GetConfigurationServer) wait.ConditionFunc {
	return func() (bool /*done*/, error) {
		config, err := a.fetchConfiguration(stream.Context(), agentInfo)
		if err != nil {
			log.WithError(err).WithField(api.AgentId, agentInfo.Id).Warn("Failed to fetch configuration")
			return false, nil // don't want to close the response stream, so report no error
		}
		return false, stream.Send(config)
	}
}

// fetchConfiguration fetches agent's configuration from a corresponding repository.
// Assumes configuration is stored in "agents/<agent id>/config.yaml" file.
func (a *Agent) fetchConfiguration(ctx context.Context, agentInfo *api.AgentInfo) (*agentrpc.ConfigurationResponse, error) {
	filename := path.Join(agentConfigurationDirectory, agentInfo.Name, agentConfigurationFileName)
	configYAML, err := a.fetchSingleFile(ctx, &agentInfo.Repository, filename, "master") // TODO handle different default branch
	if configYAML == nil {
		return nil, fmt.Errorf("configuration file not found: %q", filename)
	}
	configFile, err := parseYAMLToConfiguration(configYAML)
	if err != nil {
		return nil, fmt.Errorf("parse agent configuration: %v", err)
	}
	agentConfig, err := extractAgentConfiguration(configFile)
	if err != nil {
		return nil, fmt.Errorf("extract agent configuration: %v", err)
	}
	return &agentrpc.ConfigurationResponse{
		Configuration: agentConfig,
	}, nil
}

// fetchSingleFile fetches the latest revision of a single file.
// Returned data slice is nil if file was not found and is empty if the file is empty.
func (a *Agent) fetchSingleFile(ctx context.Context, repo *gitalypb.Repository, filename, revision string) ([]byte, error) {
	treeEntryReq := &gitalypb.TreeEntryRequest{
		Repository: repo,
		Revision:   []byte(revision),
		Path:       []byte(filename),
		Limit:      fileSizeLimit,
	}
	teResp, err := a.CommitServiceClient.TreeEntry(ctx, treeEntryReq)
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

func (a *Agent) GetObjectsToSynchronize(req *agentrpc.ObjectsToSynchronizeRequest, stream agentrpc.GitLabService_GetObjectsToSynchronizeServer) error {
	ctx := stream.Context()
	agentMeta, err := apiutil.AgentMetaFromContext(ctx)
	if err != nil {
		return err
	}
	agentInfo, err := a.GitLabClient.GetAgentInfo(ctx, agentMeta)
	if err != nil {
		return err
	}
	// TODO get period from request
	err = wait.PollImmediateUntil(15*time.Second, a.sendObjectsToSynchronize(agentInfo, stream, req.ProjectId), ctx.Done())
	if err == wait.ErrWaitTimeout {
		return nil // all good, ctx is done
	}
	return err
}

func (a *Agent) sendObjectsToSynchronize(agentInfo *api.AgentInfo, stream agentrpc.GitLabService_GetObjectsToSynchronizeServer, projectId string) wait.ConditionFunc {
	return func() (bool /*done*/, error) {
		ctx := stream.Context()
		// This call is made on each poll because:
		// - it checks that the agent's token is still valid
		// - repository location in Gitaly might have changed
		repoInfo, err := a.GitLabClient.GetProjectInfo(ctx, &agentInfo.Meta, projectId)
		if err != nil {
			log.WithError(err).WithField(api.AgentId, agentInfo.Id).Warn("Failed to get project info")
			return false, nil // don't want to close the response stream, so report no error
		}
		objects, revision, err := a.fetchObjectsToSynchronize(ctx, repoInfo)
		if err != nil {
			log.WithError(err).WithField(api.AgentId, agentInfo.Id).Warn("Failed to get objects to synchronize")
			return false, nil // don't want to close the response stream, so report no error
		}
		return false, stream.Send(&agentrpc.ObjectsToSynchronizeResponse{
			Revision: revision,
			Objects:  objects,
		})
	}
}

func (a *Agent) fetchObjectsToSynchronize(ctx context.Context, repoInfo *api.ProjectInfo) ([]*agentrpc.ObjectToSynchronize, string /* revision */, error) {
	findCommitReq := &gitalypb.FindCommitRequest{
		Repository: &repoInfo.Repository,
		Revision:   []byte("master"), // TODO handle different default branch
	}
	fcResp, err := a.CommitServiceClient.FindCommit(ctx, findCommitReq)
	if err != nil {
		return nil, "", fmt.Errorf("FindCommit: %v", err)
	}

	// TODO fetching just one file with a hardcoded name is a shortcut to cut scope
	filename := "manifest.yaml"
	manifestYAML, err := a.fetchSingleFile(ctx, &repoInfo.Repository, filename, fcResp.Commit.Id)
	if err != nil {
		return nil, "", err
	}
	if manifestYAML == nil {
		return nil, "", fmt.Errorf("manifest file not found: %q", filename)
	}
	return []*agentrpc.ObjectToSynchronize{
		{
			Object: manifestYAML,
		},
	}, fcResp.Commit.Id, nil
}

func parseYAMLToConfiguration(configYaml []byte) (*agentcfg.ConfigurationFile, error) {
	configJson, err := yaml.YAMLToJSON(configYaml)
	if err != nil {
		return nil, fmt.Errorf("YAMLToJSON: %v", err)
	}
	configFile := &agentcfg.ConfigurationFile{}
	err = protojson.Unmarshal(configJson, configFile)
	if err != nil {
		return nil, fmt.Errorf("protojson.Unmarshal: %v", err)
	}
	return configFile, nil
}

func extractAgentConfiguration(file *agentcfg.ConfigurationFile) (*agentrpc.AgentConfiguration, error) {
	return &agentrpc.AgentConfiguration{
		Deployments: file.Deployments,
	}, nil
}
