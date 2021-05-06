package usage_metrics

import (
	"fmt"
	"sync/atomic"
)

const (
	ModuleName = "usage_metrics"
)

type UsageData struct {
	Counters map[string]int64
}

type Counter interface {
	// Inc increment the counter by 1.
	Inc()
}

type counter struct {
	// n is the first element in an allocated struct to ensure 64 bit alignment for atomic access.
	n int64
}

func (c *counter) Inc() {
	atomic.AddInt64(&c.n, 1)
}

func (c *counter) get() int64 {
	return atomic.LoadInt64(&c.n)
}

func (c *counter) subtract(n int64) {
	atomic.AddInt64(&c.n, -n)
}

type UsageTrackerRegisterer interface {
	RegisterCounter(name string) Counter
}

type UsageTrackerCollector interface {
	// CloneUsageData returns collected usage data.
	// Only non-zero counters are returned.
	CloneUsageData() *UsageData
	Subtract(data *UsageData)
}

type UsageTrackerInterface interface {
	UsageTrackerRegisterer
	UsageTrackerCollector
}

type UsageTracker struct {
	counters map[string]*counter
}

func NewUsageTracker() *UsageTracker {
	return &UsageTracker{
		counters: map[string]*counter{},
	}
}

func (u *UsageTracker) RegisterCounter(name string) Counter {
	if _, exists := u.counters[name]; exists {
		panic(fmt.Errorf("counter with name %s already exists", name))
	}
	c := &counter{}
	u.counters[name] = c
	return c
}

func (u *UsageTracker) CloneUsageData() *UsageData {
	c := make(map[string]int64, len(u.counters))
	for name, counterItem := range u.counters {
		n := counterItem.get()
		if n == 0 {
			continue
		}
		c[name] = n
	}
	return &UsageData{
		Counters: c,
	}
}

func (u *UsageTracker) Subtract(data *UsageData) {
	for name, n := range data.Counters {
		u.counters[name].subtract(n)
	}
}
