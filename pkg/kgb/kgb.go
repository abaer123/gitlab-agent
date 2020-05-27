package kgb

import (
	"context"
	"fmt"
	"io"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/api"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"gitlab.com/gitlab-org/labkit/log"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"
)

type Agent struct {
	ReloadConfigurationPeriod time.Duration
	CommitServiceClient       gitalypb.CommitServiceClient
}

func (a *Agent) GetConfiguration(req *agentrpc.ConfigurationRequest, configStream agentrpc.GitLabService_GetConfigurationServer) error {
	ctx := configStream.Context()
	agentInfo, err := api.AgentInfoFromContext(ctx)
	if err != nil {
		return err
	}
	err = wait.PollImmediateUntil(a.ReloadConfigurationPeriod, a.sendConfiguration(agentInfo, configStream), ctx.Done())
	if err == wait.ErrWaitTimeout {
		return nil // all good, ctx is done
	}
	return err
}

func (a *Agent) sendConfiguration(agentInfo *api.AgentInfo, configStream agentrpc.GitLabService_GetConfigurationServer) wait.ConditionFunc {
	return func() (bool /*done*/, error) {
		config, err := a.fetchConfiguration(configStream.Context(), agentInfo)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				api.ProjectId: agentInfo.ProjectId,
				api.ClusterId: agentInfo.ClusterId,
				api.AgentId:   agentInfo.Id,
			}).Warn("Failed to fetch configuration")
			return false, nil // don't want to close the response stream, so report no error
		}
		return false, configStream.Send(config)
	}
}

// fetchConfiguration fetches agent's configuration from a corresponding repository.
// Assumes configuration is stored in "agents/<agent id>/config.yaml" file.
func (a *Agent) fetchConfiguration(ctx context.Context, agentInfo *api.AgentInfo) (*agentrpc.ConfigurationResponse, error) {
	// TODO make an API call to GitLab RoR. Send agentInfo.Id, agentInfo.ProjectId, agentInfo.Token

	repo := &gitalypb.Repository{
		StorageName:  "default",                                                                            // TODO fetch from GitLab RoR via API call
		RelativePath: "@hashed/19/58/19581e27de7ced00ff1ce50b2047e7a567c76b1cbaebabe5ef03f7c3017bb5b7.git", // TODO fetch from GitLab RoR via API call
		//GitObjectDirectory:            "",
		//GitAlternateObjectDirectories: nil,
		GlRepository:  "",               // TODO fetch from GitLab RoR via API call
		GlProjectPath: "root/test-proj", // TODO fetch from GitLab RoR via API call
	}

	//findCommitReq := &gitalypb.FindCommitRequest{
	//	Repository: repo,
	//	Revision:   []byte("master"), // TODO handle different default branch
	//}
	//fcResp, err := a.CommitServiceClient.FindCommit(ctx, findCommitReq)
	//if err != nil {
	//	return nil, err
	//}

	// mimicking lib/gitlab/gitaly_client/commit_service.rb#tree_entry
	treeEntryReq := &gitalypb.TreeEntryRequest{
		Repository: repo,
		Revision:   []byte("master"), //fcResp.Commit.Id
		Path:       []byte(fmt.Sprintf("agents/%s/config.yaml", agentInfo.Id)),
		Limit:      1024 * 1024,
	}
	teResp, err := a.CommitServiceClient.TreeEntry(ctx, treeEntryReq)
	if err != nil {
		return nil, fmt.Errorf("TreeEntry: %v", err)
	}
	var configYaml []byte
	for {
		entry, err := teResp.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("TreeEntry.Recv: %v", err)
		}
		configYaml = append(configYaml, entry.Data...)
	}
	configJson, err := yaml.YAMLToJSON(configYaml)
	if err != nil {
		return nil, fmt.Errorf("TreeEntry.YAMLToJSON: %v", err)
	}
	configFile := &agentrpc.ConfigurationFile{}
	err = protojson.Unmarshal(configJson, configFile)
	if err != nil {
		return nil, fmt.Errorf("TreeEntry.protojson.Unmarshal: %v", err)
	}
	agentConfig, err := extractAgentConfiguration(configFile)
	if err != nil {
		return nil, fmt.Errorf("extract agent configuration: %v", err)
	}
	return &agentrpc.ConfigurationResponse{
		Configuration: agentConfig,
	}, nil
}

func extractAgentConfiguration(file *agentrpc.ConfigurationFile) (*agentrpc.AgentConfiguration, error) {
	return &agentrpc.AgentConfiguration{
		SomeFeatureEnabled: file.SomeFeatureEnabled,
	}, nil
}
