package main

import (
	"gitlab.com/gitlab-org/cluster-integration/kubernetes-management-ng/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/kubernetes-management-ng/gitlab-agent/cmd/agentk/agentkapp"
)

func main() {
	cmd.Run(agentkapp.NewFromFlags)
}
