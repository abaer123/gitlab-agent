package main

import (
	"gitlab.com/ash2k/gitlab-agent/cmd"
	"gitlab.com/ash2k/gitlab-agent/cmd/agentk/agentkapp"
)

func main() {
	cmd.Run(agentkapp.NewFromFlags)
}
