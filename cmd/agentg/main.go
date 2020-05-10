package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"gitlab.com/ash2k/gitlab-agent/cmd"
	"gitlab.com/ash2k/gitlab-agent/cmd/agentg/agentgapp"
)

func main() {
	if err := run(); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		fmt.Fprintf(os.Stderr, "%#v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	cmd.CancelOnInterrupt(ctx, cancelFunc)

	return runWithContext(ctx)
}

func runWithContext(ctx context.Context) error {
	app, err := agentgapp.NewFromFlags(flag.CommandLine, os.Args[1:])
	if err != nil {
		return err
	}
	return app.Run(ctx)
}
