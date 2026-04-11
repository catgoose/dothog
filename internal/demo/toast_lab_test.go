// setup:feature:demo

package demo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToastLab_NewLab(t *testing.T) {
	lab := NewToastLab()
	settings := lab.Settings()
	assert.Equal(t, 2000, settings.RateMS)
	assert.Equal(t, 5000, settings.DismissDurMS)
	assert.Equal(t, 8, settings.StackSize)
	assert.Equal(t, 1, settings.BurstSize)
	assert.Equal(t, ToastMixBalanced, settings.SeverityMix)
	assert.False(t, settings.ReplayRecent)
	assert.False(t, lab.Paused())
}

func TestToastLab_SimTick(t *testing.T) {
	lab := NewToastLab()
	lab.UpdateSettings(func(s *ToastLabSettings) {
		s.BurstSize = 3
	})
	events := lab.SimTick()
	assert.Len(t, events, 3)
	for _, evt := range events {
		assert.NotEmpty(t, evt.ID)
		assert.NotEmpty(t, evt.Title)
		assert.NotEmpty(t, evt.Source)
		assert.NotZero(t, evt.DurationMS)
	}
	stats := lab.Stats()
	assert.Equal(t, int64(3), stats.Published)
}

func TestToastLab_Emit(t *testing.T) {
	lab := NewToastLab()
	evt := lab.Emit(ToastError)
	assert.Equal(t, ToastError, evt.Severity)
	assert.Equal(t, "operator", evt.Source)
	assert.True(t, evt.Sticky)
	assert.Equal(t, int64(1), lab.Stats().Published)
}

func TestToastLab_EventLogCap(t *testing.T) {
	lab := NewToastLab()
	lab.UpdateSettings(func(s *ToastLabSettings) {
		s.BurstSize = 10
	})
	for range 10 {
		lab.SimTick()
	}
	log := lab.EventLog()
	assert.LessOrEqual(t, len(log), 50)
}

func TestToastLab_LifecycleLogCap(t *testing.T) {
	lab := NewToastLab()
	for i := range 60 {
		lab.RecordLifecycle("entry " + string(rune('A'+i%26)))
	}
	log := lab.LifecycleLog()
	assert.LessOrEqual(t, len(log), 50)
}

func TestToastLab_TogglePause(t *testing.T) {
	lab := NewToastLab()
	assert.False(t, lab.Paused())
	p := lab.TogglePause()
	assert.True(t, p)
	assert.True(t, lab.Paused())
	p = lab.TogglePause()
	assert.False(t, p)
}

func TestToastLab_ResetStats(t *testing.T) {
	lab := NewToastLab()
	lab.SimTick()
	assert.Greater(t, lab.Stats().Published, int64(0))
	lab.ResetStats()
	assert.Equal(t, int64(0), lab.Stats().Published)
}

func TestToastLab_SeverityMix(t *testing.T) {
	lab := NewToastLab()
	lab.UpdateSettings(func(s *ToastLabSettings) {
		s.SeverityMix = ToastMixSuccessOnly
		s.BurstSize = 10
	})
	events := lab.SimTick()
	for _, evt := range events {
		assert.Equal(t, ToastSuccess, evt.Severity)
	}
}
