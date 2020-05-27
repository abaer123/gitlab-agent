package main

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd/agentg/agentgapp"
)

func main() {
	cmd.Run(agentgapp.NewFromFlags)
}
