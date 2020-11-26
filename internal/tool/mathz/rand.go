package mathz

import (
	"math/rand"
	"sync"
	"time"
)

var (
	r  = rand.New(rand.NewSource(time.Now().UnixNano())) // nolint:gosec
	mu sync.Mutex
)

func Int63n(n int64) int64 {
	mu.Lock()
	defer mu.Unlock()
	return r.Int63n(n)
}

// DurationWithJitter returns d with an added jitter between +/- jitterPercent% of the value.
func DurationWithJitter(d time.Duration, jitterPercent int64) time.Duration {
	r := (int64(d) * jitterPercent) / 100
	jitter := Int63n(2*r) - r
	return d + time.Duration(jitter)
}
