package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd/cli/generate"
)

const (
	helpText = `
Usage:
  %s [command]

Available Commands:
  generate                  Prints the YAML manifests based on specified configuration
`
)

func main() {
	cmd.Run(NewFromFlags)
}

func NewFromFlags(flagset *pflag.FlagSet, programName string, arguments []string) (cmd.Runnable, error) {
	if len(arguments) == 0 {
		return nil, fmt.Errorf("subcommand not specified\n%s", fmt.Sprintf(helpText, filepath.Base(programName)))
	}

	subcommand, args := arguments[0], arguments[1:]

	// "generate" is currently the only supported subcommand, potentially more to come
	switch subcommand {
	case "generate":
		return generate.NewFromFlags(flagset, args)
	default:
		return nil, fmt.Errorf("unknown subcommand: %s\n%s", subcommand, fmt.Sprintf(helpText, filepath.Base(programName)))
	}
}
