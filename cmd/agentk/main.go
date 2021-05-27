package main

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd/agentk/agentkapp"
)

func main() {
	cmd.Run(agentkapp.NewFromFlags)
}
