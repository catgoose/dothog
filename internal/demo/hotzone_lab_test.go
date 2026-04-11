// setup:feature:demo

package demo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHotZoneLab_NewLab(t *testing.T) {
	lab := NewHotZoneLab()
	s := lab.Settings()
	assert.Equal(t, 500, s.UpdateIntervalMS)
	assert.Equal(t, 4, s.RegionCount)
	assert.Equal(t, 100, s.PayloadSize)
	assert.Equal(t, HotZoneModeTavern, s.CommandMode)
	assert.Equal(t, HotZoneSwapInner, s.SwapScope)
	assert.Equal(t, HotZonePresetNormal, s.Preset)
	assert.False(t, s.BurstMode)
	assert.False(t, lab.Paused())
	// Heat defaults
	assert.True(t, s.HeatEnabled)
	assert.Equal(t, 1000, s.HeatWindowMS)
	assert.Equal(t, 10, s.HeatThreshold1)
	assert.Equal(t, 50, s.HeatThreshold2)
	assert.Equal(t, 100, s.HeatThreshold3)
	assert.Equal(t, "#22c55e", s.HeatColor1)
	assert.Equal(t, "#ef4444", s.HeatColor2)
	assert.Equal(t, "#a855f7", s.HeatColor3)
	assert.Equal(t, "#1e293b", s.HeatBaseColor)
}

func TestHotZoneLab_ApplyPreset(t *testing.T) {
	tests := []struct {
		preset   HotZonePreset
		interval int
		regions  int
		payload  int
		burst    bool
	}{
		{HotZonePresetNormal, 500, 4, 100, false},
		{HotZonePresetHot, 200, 6, 500, true},
		{HotZonePresetNasty, 75, 6, 1500, true},
		{HotZonePresetHell, 25, 8, 4000, true},
	}
	for _, tt := range tests {
		t.Run(string(tt.preset), func(t *testing.T) {
			lab := NewHotZoneLab()
			lab.UpdateSettings(func(s *HotZoneSettings) {
				s.ApplyPreset(tt.preset)
			})
			s := lab.Settings()
			assert.Equal(t, tt.interval, s.UpdateIntervalMS)
			assert.Equal(t, tt.regions, s.RegionCount)
			assert.Equal(t, tt.payload, s.PayloadSize)
			assert.Equal(t, tt.burst, s.BurstMode)
			assert.Equal(t, tt.preset, s.Preset)
			// All presets should set valid heat config.
			assert.True(t, s.HeatEnabled)
			assert.Greater(t, s.HeatThreshold1, 0)
			assert.Greater(t, s.HeatThreshold2, s.HeatThreshold1)
			assert.Greater(t, s.HeatThreshold3, s.HeatThreshold2)
			assert.NotEmpty(t, s.HeatColor1)
			assert.NotEmpty(t, s.HeatColor2)
			assert.NotEmpty(t, s.HeatColor3)
		})
	}
}

func TestHotZoneLab_PresetHeatThresholds(t *testing.T) {
	lab := NewHotZoneLab()
	// Nasty and Hell should have higher thresholds than Normal.
	lab.UpdateSettings(func(s *HotZoneSettings) { s.ApplyPreset(HotZonePresetNasty) })
	nasty := lab.Settings()
	assert.Equal(t, 20, nasty.HeatThreshold1)
	assert.Equal(t, 80, nasty.HeatThreshold2)
	assert.Equal(t, 150, nasty.HeatThreshold3)

	lab.UpdateSettings(func(s *HotZoneSettings) { s.ApplyPreset(HotZonePresetHell) })
	hell := lab.Settings()
	assert.Equal(t, 30, hell.HeatThreshold1)
	assert.Equal(t, 100, hell.HeatThreshold2)
	assert.Equal(t, 200, hell.HeatThreshold3)
}

func TestHotZoneLab_RecordReceived(t *testing.T) {
	lab := NewHotZoneLab()
	lab.RecordReceived(HotZoneModeTavern)
	lab.RecordReceived(HotZoneModeTavern)
	lab.RecordReceived(HotZoneModeHXPost)

	stats := lab.CommandStats()
	hx := stats[0]
	tavern := stats[1]
	assert.Equal(t, int64(1), hx.Received)
	assert.Equal(t, int64(0), hx.Dispatched)
	assert.Equal(t, int64(2), tavern.Received)
	assert.Equal(t, int64(0), tavern.Dispatched)
}

func TestHotZoneLab_RecordLifecycle(t *testing.T) {
	lab := NewHotZoneLab()

	// Simulate a full tavern command lifecycle.
	lab.RecordLifecycle(HotZoneModeTavern, "dispatched")
	lab.RecordLifecycle(HotZoneModeTavern, "succeeded")

	// Simulate a failed hx-post command.
	lab.RecordLifecycle(HotZoneModeHXPost, "dispatched")
	lab.RecordLifecycle(HotZoneModeHXPost, "failed")

	stats := lab.CommandStats()
	hx := stats[0]
	tavern := stats[1]

	assert.Equal(t, int64(1), tavern.Dispatched)
	assert.Equal(t, int64(1), tavern.Succeeded)
	assert.Equal(t, int64(0), tavern.Failed)

	assert.Equal(t, int64(1), hx.Dispatched)
	assert.Equal(t, int64(0), hx.Succeeded)
	assert.Equal(t, int64(1), hx.Failed)
}

func TestHotZoneLab_StatsIndependent(t *testing.T) {
	lab := NewHotZoneLab()

	// One command: client dispatches, server receives, client succeeds.
	lab.RecordLifecycle(HotZoneModeTavern, "dispatched")
	lab.RecordReceived(HotZoneModeTavern)
	lab.RecordLifecycle(HotZoneModeTavern, "succeeded")

	stats := lab.CommandStats()
	tavern := stats[1]
	assert.Equal(t, int64(1), tavern.Dispatched)
	assert.Equal(t, int64(1), tavern.Received)
	assert.Equal(t, int64(1), tavern.Succeeded)
	assert.Equal(t, int64(0), tavern.Failed)
}

func TestHotZoneLab_ResetStats(t *testing.T) {
	lab := NewHotZoneLab()
	lab.RecordLifecycle(HotZoneModeTavern, "dispatched")
	lab.RecordReceived(HotZoneModeTavern)
	lab.RecordLifecycle(HotZoneModeTavern, "succeeded")

	lab.ResetStats()
	stats := lab.CommandStats()
	for _, s := range stats {
		assert.Equal(t, int64(0), s.Dispatched)
		assert.Equal(t, int64(0), s.Received)
		assert.Equal(t, int64(0), s.Succeeded)
		assert.Equal(t, int64(0), s.Failed)
	}
}

func TestHotZoneLab_SimTick(t *testing.T) {
	lab := NewHotZoneLab()
	updated := lab.SimTick()
	assert.Len(t, updated, 1)
	assert.GreaterOrEqual(t, updated[0], 1)
	assert.LessOrEqual(t, updated[0], 4)
}

func TestHotZoneLab_BurstMode(t *testing.T) {
	lab := NewHotZoneLab()
	lab.UpdateSettings(func(s *HotZoneSettings) {
		s.BurstMode = true
		s.RegionCount = 3
	})
	updated := lab.SimTick()
	assert.Len(t, updated, 3)
}
