package main

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd/kas/kasapp"
)

func main() {
	cmd.Run(kasapp.NewFromFlags)
}
