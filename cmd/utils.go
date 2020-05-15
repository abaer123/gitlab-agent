package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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

type RunnableFactory func(flagset *flag.FlagSet, arguments []string) (Runnable, error)

func Run(factory RunnableFactory) {
	if err := run(factory); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		fmt.Fprintf(os.Stderr, "%#v\n", err)
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
	app, err := factory(flag.CommandLine, os.Args[1:])
	if err != nil {
		return err
	}
	return app.Run(ctx)
}
