package cmd

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/labkit/log"
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
	if err := run(factory); err != nil && err != context.Canceled && err != context.DeadlineExceeded && err != pflag.ErrHelp {
		log.WithError(err).Error("Program aborted")
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
	flagset := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	app, err := factory(flagset, os.Args[1:])
	if err != nil {
		return err
	}
	return app.Run(ctx)
}
