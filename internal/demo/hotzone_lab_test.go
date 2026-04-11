// setup:feature:demo

package demo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testPool() HotZoneImagePool {
	return HotZoneImagePool{
		All: []HotZoneImage{
			{Path: "/public/images/hotzones/anime/nge-rei.jpg", Alt: "rei", Series: "nge", IsGIF: false},
			{Path: "/public/images/hotzones/anime/flcl-haruko.png", Alt: "haruko", Series: "flcl", IsGIF: false},
			{Path: "/public/images/hotzones/anime/trigun-vash.gif", Alt: "vash", Series: "trigun", IsGIF: true},
		},
		Static: []HotZoneImage{
			{Path: "/public/images/hotzones/anime/nge-rei.jpg", Alt: "rei", Series: "nge", IsGIF: false},
			{Path: "/public/images/hotzones/anime/flcl-haruko.png", Alt: "haruko", Series: "flcl", IsGIF: false},
		},
	}
}

func TestHotZoneLab_NewLab(t *testing.T) {
	lab := NewHotZoneLab(testPool())
	s := lab.Settings()
	assert.Equal(t, 500, s.UpdateIntervalMS)
	assert.Equal(t, 4, s.RegionCount)
	assert.Equal(t, HotZoneModeTavern, s.CommandMode)
	assert.Equal(t, HotZoneSwapInner, s.SwapScope)
	assert.Equal(t, HotZonePresetNormal, s.Preset)
	assert.False(t, s.BurstMode)
	assert.True(t, s.AllowGIF)
	assert.True(t, s.ShowMeta)
	assert.Equal(t, 100, s.JitterMinMS)
	assert.Equal(t, 500, s.JitterMaxMS)
	assert.False(t, lab.Paused())
	// Heat defaults
	assert.True(t, s.HeatEnabled)
	assert.Equal(t, 1000, s.HeatWindowMS)
	assert.Equal(t, 8, s.HeatThreshold1)
	assert.Equal(t, 16, s.HeatThreshold2)
	assert.Equal(t, 32, s.HeatThreshold3)
	// Regions initialized with images
	regions := lab.Regions()
	assert.Len(t, regions, 4)
	for _, r := range regions {
		assert.NotEmpty(t, r.ImagePath)
	}
}

func TestHotZoneLab_ApplyPreset(t *testing.T) {
	tests := []struct {
		preset   HotZonePreset
		interval int
		regions  int
		burst    bool
	}{
		{HotZonePresetNormal, 500, 4, false},
		{HotZonePresetHot, 200, 8, true},
		{HotZonePresetNasty, 75, 16, true},
		{HotZonePresetHell, 25, 32, true},
	}
	for _, tt := range tests {
		t.Run(string(tt.preset), func(t *testing.T) {
			lab := NewHotZoneLab(testPool())
			lab.UpdateSettings(func(s *HotZoneSettings) {
				s.ApplyPreset(tt.preset)
			})
			s := lab.Settings()
			assert.Equal(t, tt.interval, s.UpdateIntervalMS)
			assert.Equal(t, tt.regions, s.RegionCount)
			assert.Equal(t, tt.burst, s.BurstMode)
			assert.Equal(t, tt.preset, s.Preset)
			// All presets set valid heat config.
			assert.True(t, s.HeatEnabled)
			assert.Greater(t, s.HeatThreshold1, 0)
			assert.Greater(t, s.HeatThreshold2, s.HeatThreshold1)
			assert.Greater(t, s.HeatThreshold3, s.HeatThreshold2)
		})
	}
}

func TestHotZoneLab_PresetHeatThresholds(t *testing.T) {
	lab := NewHotZoneLab(testPool())
	// All presets use the same default heat thresholds.
	for _, p := range []HotZonePreset{HotZonePresetNormal, HotZonePresetHot, HotZonePresetNasty, HotZonePresetHell} {
		lab.UpdateSettings(func(s *HotZoneSettings) { s.ApplyPreset(p) })
		s := lab.Settings()
		assert.Equal(t, 8, s.HeatThreshold1, "preset %s", p)
		assert.Equal(t, 16, s.HeatThreshold2, "preset %s", p)
		assert.Equal(t, 32, s.HeatThreshold3, "preset %s", p)
	}
}

func TestHotZoneLab_ToggleLock(t *testing.T) {
	lab := NewHotZoneLab(testPool())
	// Initially unlocked.
	r := lab.Region(1)
	assert.False(t, r.Locked)
	// Lock.
	locked := lab.ToggleLock(1)
	assert.True(t, locked)
	r = lab.Region(1)
	assert.True(t, r.Locked)
	// Unlock.
	locked = lab.ToggleLock(1)
	assert.False(t, locked)
	r = lab.Region(1)
	assert.False(t, r.Locked)
}

func TestHotZoneLab_SimTickSkipsLocked(t *testing.T) {
	lab := NewHotZoneLab(testPool())
	lab.UpdateSettings(func(s *HotZoneSettings) {
		s.RegionCount = 1
		s.FocusedRegion = 1
	})
	lab.ToggleLock(1)
	updated := lab.SimTick()
	assert.Empty(t, updated, "locked region should not be updated")
}

func TestHotZoneLab_SimTickUpdatesUnlocked(t *testing.T) {
	lab := NewHotZoneLab(testPool())
	lab.UpdateSettings(func(s *HotZoneSettings) {
		s.RegionCount = 1
		s.FocusedRegion = 1
	})
	updated := lab.SimTick()
	assert.Len(t, updated, 1)
	assert.Equal(t, 1, updated[0])
}

func TestHotZoneLab_BurstSkipsLocked(t *testing.T) {
	lab := NewHotZoneLab(testPool())
	lab.UpdateSettings(func(s *HotZoneSettings) {
		s.BurstMode = true
		s.RegionCount = 3
	})
	lab.ToggleLock(2)
	updated := lab.SimTick()
	assert.Len(t, updated, 2)
	for _, id := range updated {
		assert.NotEqual(t, 2, id)
	}
}

func TestHotZoneLab_JitteredInterval(t *testing.T) {
	lab := NewHotZoneLab(testPool())
	lab.UpdateSettings(func(s *HotZoneSettings) {
		s.UpdateIntervalMS = 100
		s.JitterMinMS = 50
		s.JitterMaxMS = 200
	})
	d := lab.JitteredInterval()
	// Should be between 150ms and 300ms (100 + 50..200).
	assert.GreaterOrEqual(t, d.Milliseconds(), int64(150))
	assert.LessOrEqual(t, d.Milliseconds(), int64(300))
}

func TestHotZoneLab_RecordReceived(t *testing.T) {
	lab := NewHotZoneLab(testPool())
	lab.RecordReceived(HotZoneModeTavern)
	lab.RecordReceived(HotZoneModeTavern)
	lab.RecordReceived(HotZoneModeHXPost)
	stats := lab.CommandStats()
	assert.Equal(t, int64(1), stats[0].Received)
	assert.Equal(t, int64(2), stats[1].Received)
}

func TestHotZoneLab_ResetStats(t *testing.T) {
	lab := NewHotZoneLab(testPool())
	lab.RecordReceived(HotZoneModeTavern)
	lab.ResetStats()
	stats := lab.CommandStats()
	for _, s := range stats {
		assert.Equal(t, int64(0), s.Dispatched)
		assert.Equal(t, int64(0), s.Received)
		assert.Equal(t, int64(0), s.Succeeded)
		assert.Equal(t, int64(0), s.Failed)
	}
}
