package retry

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

func JitterUntil(ctx context.Context, period time.Duration, f func(context.Context)) {
	wait.JitterUntilWithContext(ctx, f, period, 1.5, true)
}
