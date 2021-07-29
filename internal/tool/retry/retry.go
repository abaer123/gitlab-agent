package retry

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"
)

type AttemptResult int

const (
	// Continue means there was no error and polling can continue normally.
	Continue AttemptResult = iota
	// ContinueImmediately means there was no error and polling can continue normally.
	ContinueImmediately
	// Backoff means there was a retriable error, so the caller should try later.
	Backoff
	// Done means the polling should stop. There may or may not have been an error.
	Done
)

var (
	ErrWaitTimeout = wait.ErrWaitTimeout
)

type ConditionFunc = wait.ConditionFunc
type BackoffManager = wait.BackoffManager
type BackoffManagerFactory func() BackoffManager

// PollWithBackoffFunc is a function that is called to perform polling.
// Signature is unusual because AttemptResult must be checked, not the error.
type PollWithBackoffFunc func() (error, AttemptResult)

type PollConfig struct {
	Backoff  BackoffManager
	Interval time.Duration
	Sliding  bool
}

type PollConfigFactory func() PollConfig

// PollImmediateUntil is a wrapper to make the function more convenient to use.
// - ctx is used instead of a channel.
// - ctx is the first argument to follow the convention.
// - condition is the last argument because code is more readable this way when used with inline functions.
func PollImmediateUntil(ctx context.Context, interval time.Duration, f ConditionFunc) error {
	return wait.PollImmediateUntil(interval, f, ctx.Done())
}

// PollWithBackoff runs f every duration given by BackoffManager.
//
// If sliding is true, the period is computed after f runs. If it is false then
// period includes the runtime for f.
// It returns when:
// - context signals done. ErrWaitTimeout is returned in this case.
// - f returns Done
func PollWithBackoff(ctx context.Context, cfg PollConfig, f PollWithBackoffFunc) error {
	var t clock.Timer
	defer func() {
		if t != nil && !t.Stop() {
			<-t.C()
		}
	}()
	doneCh := ctx.Done()
	for {
		if !cfg.Sliding {
			t = cfg.Backoff.Backoff()
		}

	attempt:
		for {
			select {
			case <-doneCh:
				return ErrWaitTimeout
			default:
			}
			err, result := f()
			switch result {
			case Continue: // sleep and continue
				timer := time.NewTimer(cfg.Interval)
				select {
				case <-doneCh:
					timer.Stop()
					return ErrWaitTimeout
				case <-timer.C:
				}
			case ContinueImmediately: // immediately call f again
				continue
			case Backoff: // do an outer loop to backoff
				break attempt
			case Done: // f is done. A success or a terminal failure.
				return err
			default:
				panic(fmt.Errorf("unexpected poll attempt result: %v", result))
			}
		}

		if cfg.Sliding {
			t = cfg.Backoff.Backoff()
		}

		// NOTE: b/c there is no priority selection in golang
		// it is possible for this to race, meaning we could
		// trigger t.C and stopCh, and t.C select falls through.
		// In order to mitigate we re-check stopCh at the beginning
		// of every loop to prevent extra executions of f().
		select {
		case <-doneCh:
			return ErrWaitTimeout
		case <-t.C():
			t = nil
		}
	}
}

func NewExponentialBackoffFactory(initBackoff, maxBackoff, resetDuration time.Duration, backoffFactor, jitter float64) BackoffManagerFactory {
	return func() BackoffManager {
		return wait.NewExponentialBackoffManager(initBackoff, maxBackoff, resetDuration, backoffFactor, jitter, clock.RealClock{})
	}
}

func NewPollConfigFactory(interval time.Duration, backoff BackoffManagerFactory) PollConfigFactory {
	return func() PollConfig {
		return PollConfig{
			Backoff:  backoff(),
			Interval: interval,
			Sliding:  true,
		}
	}
}
