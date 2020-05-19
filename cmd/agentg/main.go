package main

import (
	"gitlab.com/gitlab-org/cluster-integration/kubernetes-management-ng/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/kubernetes-management-ng/gitlab-agent/cmd/agentg/agentgapp"
)

func main() {
	cmd.Run(agentgapp.NewFromFlags)
}
