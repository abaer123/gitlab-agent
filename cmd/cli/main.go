package main

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd/cli/generate"
)

func main() {
	cmd.Run(rootCommand())
}

func rootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "cli",
		Short:         "GitLab Kubernetes Agent CLI",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCmd.AddCommand(generate.NewCommand())
	return rootCmd
}
