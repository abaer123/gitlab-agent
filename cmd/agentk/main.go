package main

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd/agentk/agentkapp"
)

func main() {
	cmd.Run(agentkapp.NewFromFlags)
}
