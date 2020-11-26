package retry

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

func JitterUntil(ctx context.Context, period time.Duration, f func(context.Context)) {
	wait.JitterUntilWithContext(ctx, f, period, 1.5, true)
}

// PollImmediateUntil is a wrapper to make the function more convenient to use.
// - ctx is used instead of a channel.
// - ctx is the first argument to follow the convention.
// - condition is the last argument because code is more readable this way when used with inline functions.
func PollImmediateUntil(ctx context.Context, interval time.Duration, condition wait.ConditionFunc) error {
	return wait.PollImmediateUntil(interval, condition, ctx.Done())
}
