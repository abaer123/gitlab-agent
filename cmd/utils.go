package cmd

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ash2k/stager"
	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/errz"
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

type Runnable interface {
	Run(context.Context) error
}

type RunnableFactory func(flagset *pflag.FlagSet, programName string, arguments []string) (Runnable, error)

func Run(factory RunnableFactory) {
	rand.Seed(time.Now().UnixNano())
	if err := run(factory); err != nil && !errz.ContextDone(err) && !errors.Is(err, pflag.ErrHelp) {
		fmt.Fprintf(os.Stderr, "Program aborted: %v\n", err)
		os.Exit(1)
	}
}

func run(factory RunnableFactory) error {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelFunc()

	return runWithContext(ctx, factory)
}

func runWithContext(ctx context.Context, factory RunnableFactory) error {
	programName := os.Args[0]
	binaryName := filepath.Base(programName)
	flagset := pflag.NewFlagSet(binaryName, pflag.ContinueOnError)
	printVersion := flagset.Bool("version", false, "Print version and exit")
	app, err := factory(flagset, programName, os.Args[1:])
	if err != nil {
		return err
	}
	if *printVersion {
		fmt.Fprintf(os.Stderr, "%s version: %s, commit: %s, built: %s\n", binaryName, Version, Commit, BuildTime)
		return nil
	}
	return app.Run(ctx)
}
