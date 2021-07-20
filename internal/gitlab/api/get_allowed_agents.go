package api

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
)

const (
	AllowedAgentsApiPath = "/api/v4/job/allowed_agents"
)

type AllowedAgent struct {
	Id            int64         `json:"id"`
	ConfigProject ConfigProject `json:"config_project"`
}

type ConfigProject struct {
	Id int64 `json:"id"`
}

type Pipeline struct {
	Id int64 `json:"id"`
}

type Project struct {
	Id int64 `json:"id"`
}

type Job struct {
	Id int64 `json:"id"`
}

type User struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
}

type AllowedAgentsForJob struct {
	AllowedAgents []AllowedAgent `json:"allowed_agents"`
	Job           Job            `json:"job"`
	Pipeline      Pipeline       `json:"pipeline"`
	Project       Project        `json:"project"`
	User          User           `json:"user"`
}

func GetAllowedAgentsForJob(ctx context.Context, client gitlab.ClientInterface, jobToken string) (*AllowedAgentsForJob, error) {
	ji := &AllowedAgentsForJob{}
	err := client.Do(ctx,
		gitlab.WithPath(AllowedAgentsApiPath),
		gitlab.WithJobToken(jobToken),
		gitlab.WithResponseHandler(gitlab.JsonResponseHandler(ji)),
	)
	if err != nil {
		return nil, err
	}
	return ji, nil
}
