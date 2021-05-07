package usage_metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	_ Counter               = &counter{}
	_ UsageTrackerInterface = &UsageTracker{}
)

func TestUsageTracker(t *testing.T) {
	u := NewUsageTracker()
	c := u.RegisterCounter("x")
	require.Contains(t, u.counters, "x")

	ud := u.CloneUsageData()
	expected := map[string]int64{}
	require.Equal(t, expected, ud.Counters)

	c.Inc()
	ud = u.CloneUsageData()
	expected = map[string]int64{
		"x": 1,
	}
	require.Equal(t, expected, ud.Counters)

	u.Subtract(ud)
	ud = u.CloneUsageData()
	expected = map[string]int64{}
	require.Equal(t, expected, ud.Counters)
}
