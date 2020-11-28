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

	ud, allZeroes := u.CloneUsageData()
	expected := map[string]int64{
		"x": 0,
	}
	require.Equal(t, expected, ud.Counters)
	require.True(t, allZeroes)

	c.Inc()
	ud, allZeroes = u.CloneUsageData()
	expected = map[string]int64{
		"x": 1,
	}
	require.Equal(t, expected, ud.Counters)
	require.False(t, allZeroes)

	u.Subtract(ud)
	ud, allZeroes = u.CloneUsageData()
	expected = map[string]int64{
		"x": 0,
	}
	require.Equal(t, expected, ud.Counters)
	require.True(t, allZeroes)
}
