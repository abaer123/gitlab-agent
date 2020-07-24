package main

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd/kas/kasapp"
)

func main() {
	cmd.Run(kasapp.NewFromFlags)
}
