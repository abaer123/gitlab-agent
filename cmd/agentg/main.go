package main

import (
	"gitlab.com/ash2k/gitlab-agent/cmd"
	"gitlab.com/ash2k/gitlab-agent/cmd/agentg/agentgapp"
)

func main() {
	cmd.Run(agentgapp.NewFromFlags)
}
