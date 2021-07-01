package main

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd/agentk/agentkapp"

	// Install client auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	cmd.Run(agentkapp.NewCommand())
}
