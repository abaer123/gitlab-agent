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

	"github.com/spf13/pflag"
)

// CancelOnInterrupt calls f when os.Interrupt or SIGTERM is received.
// It ignores subsequent interrupts on purpose - program should exit correctly after the first signal.
func CancelOnInterrupt(ctx context.Context, f context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-ctx.Done():
		case <-c:
			f()
		}
	}()
}

type Runnable interface {
	Run(context.Context) error
}

type RunnableFactory func(flagset *pflag.FlagSet, arguments []string) (Runnable, error)

func Run(factory RunnableFactory) {
	rand.Seed(time.Now().UnixNano())
	if err := run(factory); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, pflag.ErrHelp) {
		fmt.Fprintf(os.Stderr, "Program aborted: %v\n", err)
		os.Exit(1)
	}
}

func run(factory RunnableFactory) error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	CancelOnInterrupt(ctx, cancelFunc)

	return runWithContext(ctx, factory)
}

func runWithContext(ctx context.Context, factory RunnableFactory) error {
	programName := os.Args[0]
	flagset := pflag.NewFlagSet(programName, pflag.ContinueOnError)
	printVersion := flagset.Bool("version", false, "Print version and exit")
	app, err := factory(flagset, os.Args[1:])
	if err != nil {
		return err
	}
	if *printVersion {
		fmt.Fprintf(os.Stderr, "%s version: %s, commit: %s\n", filepath.Base(programName), Version, Commit)
		return nil
	}
	return app.Run(ctx)
}
