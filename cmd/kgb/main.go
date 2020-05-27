package main

import (
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd/kgb/kgbapp"
)

func main() {
	cmd.Run(kgbapp.NewFromFlags)
}
