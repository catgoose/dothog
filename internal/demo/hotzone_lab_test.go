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
	assert.Equal(t, 8, s.GridSize)
	assert.Equal(t, HotZoneModeTavern, s.CommandMode)
	assert.Equal(t, HotZoneSwapInner, s.SwapScope)
	assert.Equal(t, HotZonePresetNormal, s.Preset)
	assert.Equal(t, 100, s.JitterMinMS)
	assert.Equal(t, 500, s.JitterMaxMS)
	assert.False(t, s.BurstMode)
	assert.False(t, lab.Paused())
	// Heat defaults
	assert.True(t, s.HeatEnabled)
	assert.Equal(t, 8, s.HeatThreshold1)
	assert.Equal(t, 16, s.HeatThreshold2)
	assert.Equal(t, 32, s.HeatThreshold3)
	// Regions initialized with cells
	regions := lab.Regions()
	assert.Len(t, regions, 4)
	for _, r := range regions {
		assert.Len(t, r.Cells, 64) // 8x8
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
			lab := NewHotZoneLab()
			lab.UpdateSettings(func(s *HotZoneSettings) {
				s.ApplyPreset(tt.preset)
			})
			s := lab.Settings()
			assert.Equal(t, tt.interval, s.UpdateIntervalMS)
			assert.Equal(t, tt.regions, s.RegionCount)
			assert.Equal(t, tt.burst, s.BurstMode)
			assert.Equal(t, tt.preset, s.Preset)
			assert.Equal(t, 8, s.GridSize)
		})
	}
}

func TestHotZoneLab_CellCount(t *testing.T) {
	lab := NewHotZoneLab()
	lab.UpdateSettings(func(s *HotZoneSettings) {
		s.GridSize = 4
	})
	lab.SimTick()
	r := lab.Region(1)
	// After a tick with GridSize=4, cells should be 4*4=16.
	if len(r.Cells) > 0 {
		// The region was updated by SimTick.
		assert.Len(t, r.Cells, 16)
	}
}

func TestHotZoneLab_ToggleLock(t *testing.T) {
	lab := NewHotZoneLab()
	assert.False(t, lab.Region(1).Locked)
	assert.True(t, lab.ToggleLock(1))
	assert.True(t, lab.Region(1).Locked)
	assert.False(t, lab.ToggleLock(1))
	assert.False(t, lab.Region(1).Locked)
}

func TestHotZoneLab_SimTickSkipsLocked(t *testing.T) {
	lab := NewHotZoneLab()
	lab.UpdateSettings(func(s *HotZoneSettings) {
		s.RegionCount = 1
		s.FocusedRegion = 1
	})
	lab.ToggleLock(1)
	assert.Empty(t, lab.SimTick())
}

func TestHotZoneLab_BurstSkipsLocked(t *testing.T) {
	lab := NewHotZoneLab()
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
	lab := NewHotZoneLab()
	lab.UpdateSettings(func(s *HotZoneSettings) {
		s.UpdateIntervalMS = 100
		s.JitterMinMS = 50
		s.JitterMaxMS = 200
	})
	d := lab.JitteredInterval()
	assert.GreaterOrEqual(t, d.Milliseconds(), int64(150))
	assert.LessOrEqual(t, d.Milliseconds(), int64(300))
}

func TestHotZoneLab_ResetStats(t *testing.T) {
	lab := NewHotZoneLab()
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

func TestHotZoneLab_PaletteValid(t *testing.T) {
	lab := NewHotZoneLab()
	regions := lab.Regions()
	for _, r := range regions {
		for _, c := range r.Cells {
			assert.GreaterOrEqual(t, c.Palette, 0)
			assert.Less(t, c.Palette, len(HotZonePaletteColors))
			assert.NotEmpty(t, c.Glyph)
		}
	}
}
