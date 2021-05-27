package agentkapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modagent"
	"go.uber.org/zap/zaptest"
)

func TestFeatureTracker_FirstEnableCallsCallback(t *testing.T) {
	ft := newFeatureTracker(zaptest.NewLogger(t))
	called := 0
	ft.Subscribe(modagent.Tunnel, func(enabled bool) {
		assert.True(t, enabled)
		called++
	})
	ft.ToggleFeature(modagent.Tunnel, "bla", true)
	assert.EqualValues(t, 1, called)
}

func TestFeatureTracker_MultipleEnableCallCallbackOnce(t *testing.T) {
	t.Run("same consumer", func(t *testing.T) {
		ft := newFeatureTracker(zaptest.NewLogger(t))
		called := 0
		ft.Subscribe(modagent.Tunnel, func(enabled bool) {
			assert.True(t, enabled)
			called++
		})
		ft.ToggleFeature(modagent.Tunnel, "bla", true)
		ft.ToggleFeature(modagent.Tunnel, "bla", true)
		assert.EqualValues(t, 1, called)
	})
	t.Run("different consumers", func(t *testing.T) {
		ft := newFeatureTracker(zaptest.NewLogger(t))
		called := 0
		ft.Subscribe(modagent.Tunnel, func(enabled bool) {
			assert.True(t, enabled)
			called++
		})
		ft.ToggleFeature(modagent.Tunnel, "bla1", true)
		ft.ToggleFeature(modagent.Tunnel, "bla2", true)
		assert.EqualValues(t, 1, called)
	})
}

func TestFeatureTracker_DisableIsCalledOnce(t *testing.T) {
	t.Run("same consumer", func(t *testing.T) {
		ft := newFeatureTracker(zaptest.NewLogger(t))
		called := 0
		ft.ToggleFeature(modagent.Tunnel, "bla", true)
		ft.Subscribe(modagent.Tunnel, func(enabled bool) {
			assert.False(t, enabled)
			called++
		})
		ft.ToggleFeature(modagent.Tunnel, "bla", false)
		assert.EqualValues(t, 1, called)
		ft.ToggleFeature(modagent.Tunnel, "bla", false)
		assert.EqualValues(t, 1, called) // still one
	})
	t.Run("different consumers", func(t *testing.T) {
		ft := newFeatureTracker(zaptest.NewLogger(t))
		called := 0
		ft.ToggleFeature(modagent.Tunnel, "bla1", true)
		ft.ToggleFeature(modagent.Tunnel, "bla2", true)
		ft.Subscribe(modagent.Tunnel, func(enabled bool) {
			assert.False(t, enabled)
			called++
		})
		ft.ToggleFeature(modagent.Tunnel, "bla1", false)
		assert.Zero(t, called)
		ft.ToggleFeature(modagent.Tunnel, "bla2", false)
		assert.EqualValues(t, 1, called)
	})
}
