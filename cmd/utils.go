package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ash2k/stager"
	"github.com/spf13/cobra"
)

// StageFunc is a function that uses the provided Stage to start goroutines.
type StageFunc func(stager.Stage)

// RunStages is a helper that ensures Run() is always executed and there is no chance of early exit so that
// goroutines from stages don't leak.
func RunStages(ctx context.Context, stages ...StageFunc) error {
	stgr := stager.New()
	for _, s := range stages {
		s(stgr.NextStage())
	}
	return stgr.Run(ctx)
}

func Run(command *cobra.Command) {
	rand.Seed(time.Now().UnixNano())
	command.Version = fmt.Sprintf("%s, commit: %s, built: %s", Version, Commit, BuildTime)
	err := run(command)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Program aborted: %v\n", err)
		os.Exit(1)
	}
}

func run(command *cobra.Command) error {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelFunc()

	return command.ExecuteContext(ctx)
}
