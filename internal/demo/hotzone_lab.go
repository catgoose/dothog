// setup:feature:demo

package demo

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"
)

// HotZoneMode describes how a region handles user commands.
type HotZoneMode string

const (
	HotZoneModeHXPost HotZoneMode = "hx-post"
	HotZoneModeTavern HotZoneMode = "tavern-command"
)

// HotZoneCell represents one cell in a region's text grid.
type HotZoneCell struct {
	Glyph    string
	Palette  int // index into the color palette (0–7)
}

// HotZoneRegion is a single independently-updating UI region.
type HotZoneRegion struct {
	LastUpdate time.Time
	Label      string
	Cells      []HotZoneCell
	Counter    int64
	ID         int
	Locked     bool
}

// HotZoneSwapScope controls the granularity of SSE replacement.
type HotZoneSwapScope string

const (
	HotZoneSwapInner HotZoneSwapScope = "inner"
	HotZoneSwapCard  HotZoneSwapScope = "card"
)

// HotZonePreset is a named pressure profile.
type HotZonePreset string

const (
	HotZonePresetNormal HotZonePreset = "normal"
	HotZonePresetHot    HotZonePreset = "hot"
	HotZonePresetNasty  HotZonePreset = "nasty"
	HotZonePresetHell   HotZonePreset = "hell"
)

// HotZonePalette defines bg/fg pairs for text grid cells.
// Each entry is [background, foreground] — foreground is a
// high-contrast complement of the background.
var HotZonePalette = [8][2]string{
	{"#22c55e", "#0a3d1e"}, // green bg, dark green fg
	{"#06b6d4", "#042f38"}, // cyan bg, dark cyan fg
	{"#f59e0b", "#4a2f00"}, // amber bg, dark amber fg
	{"#d946ef", "#3d0f47"}, // magenta bg, dark magenta fg
	{"#ef4444", "#4a0e0e"}, // red bg, dark red fg
	{"#3b82f6", "#0f2554"}, // blue bg, dark blue fg
	{"#1e293b", "#cbd5e1"}, // dark bg, light fg
	{"#e2e8f0", "#1e293b"}, // light bg, dark fg
}

var glyphs = []string{"█", "■", "●", "+", "▓", "◆", "▲", "◉"}

// HotZoneSettings holds operator-controlled simulation parameters.
type HotZoneSettings struct {
	Preset           HotZonePreset
	CommandMode      HotZoneMode
	SwapScope        HotZoneSwapScope
	UpdateIntervalMS int // ms between ticks (25–5000)
	RegionCount      int // 1–64
	GridSize         int // N for NxN grid (2–16)
	FocusedRegion    int // 0 = random, 1–64 = focused
	JitterMinMS      int // 0–2000
	JitterMaxMS      int // 0–5000
	BurstMode        bool
	// Heat-map visualization (client-side).
	HeatEnabled    bool
	HeatWindowMS   int
	HeatThreshold1 int
	HeatThreshold2 int
	HeatThreshold3 int
	HeatColor1     string
	HeatColor2     string
	HeatColor3     string
	HeatBaseColor  string
}

// DefaultHeatSettings returns sensible heat-map defaults.
func DefaultHeatSettings() (int, int, int, int, string, string, string, string) {
	return 1000, 8, 16, 32, "#22c55e", "#ef4444", "#a855f7", "#1e293b"
}

// ApplyPreset sets fields to the named preset's values.
func (s *HotZoneSettings) ApplyPreset(p HotZonePreset) {
	s.Preset = p
	s.HeatWindowMS, s.HeatThreshold1, s.HeatThreshold2, s.HeatThreshold3,
		s.HeatColor1, s.HeatColor2, s.HeatColor3, s.HeatBaseColor = DefaultHeatSettings()
	s.HeatEnabled = true
	switch p {
	case HotZonePresetHot:
		s.UpdateIntervalMS = 200
		s.RegionCount = 8
		s.GridSize = 8
		s.JitterMinMS = 0
		s.JitterMaxMS = 200
		s.BurstMode = true
		s.FocusedRegion = 0
	case HotZonePresetNasty:
		s.UpdateIntervalMS = 75
		s.RegionCount = 16
		s.GridSize = 8
		s.JitterMinMS = 0
		s.JitterMaxMS = 100
		s.BurstMode = true
		s.FocusedRegion = 0
	case HotZonePresetHell:
		s.UpdateIntervalMS = 25
		s.RegionCount = 32
		s.GridSize = 8
		s.JitterMinMS = 0
		s.JitterMaxMS = 50
		s.BurstMode = true
		s.FocusedRegion = 0
	default: // normal
		s.UpdateIntervalMS = 500
		s.RegionCount = 4
		s.GridSize = 8
		s.JitterMinMS = 100
		s.JitterMaxMS = 500
		s.BurstMode = false
		s.FocusedRegion = 0
	}
}

// HotZoneCommandStat tracks command lifecycle metrics per mode.
type HotZoneCommandStat struct {
	Mode       HotZoneMode
	Dispatched int64
	Received   int64
	Succeeded  int64
	Failed     int64
}

// HotZoneLab wraps the shared state for the hot-zone stress surface.
type HotZoneLab struct {
	activity         []HotZoneActivity
	regions          []HotZoneRegion
	settings         HotZoneSettings
	hxDispatched     atomic.Int64
	hxReceived       atomic.Int64
	hxOK             atomic.Int64
	hxFail           atomic.Int64
	tavernDispatched atomic.Int64
	tavernReceived   atomic.Int64
	tavernOK         atomic.Int64
	tavernFail       atomic.Int64
	mu               sync.RWMutex
	paused           bool
}

// HotZoneActivity records one action in the activity log.
type HotZoneActivity struct {
	Timestamp time.Time
	Action    string
}

// NewHotZoneLab creates a lab with default settings.
func NewHotZoneLab() *HotZoneLab {
	wMS, t1, t2, t3, c1, c2, c3, base := DefaultHeatSettings()
	lab := &HotZoneLab{
		settings: HotZoneSettings{
			Preset:           HotZonePresetNormal,
			UpdateIntervalMS: 500,
			RegionCount:      4,
			GridSize:         8,
			JitterMinMS:      100,
			JitterMaxMS:      500,
			BurstMode:        false,
			FocusedRegion:    0,
			CommandMode:      HotZoneModeTavern,
			SwapScope:        HotZoneSwapInner,
			HeatEnabled:      true,
			HeatWindowMS:     wMS,
			HeatThreshold1:   t1,
			HeatThreshold2:   t2,
			HeatThreshold3:   t3,
			HeatColor1:       c1,
			HeatColor2:       c2,
			HeatColor3:       c3,
			HeatBaseColor:    base,
		},
	}
	lab.regions = make([]HotZoneRegion, 64)
	for i := range lab.regions {
		lab.regions[i] = HotZoneRegion{
			ID:    i + 1,
			Label: fmt.Sprintf("Region %d", i+1),
		}
		lab.regions[i].Cells = generateCells(lab.settings.GridSize)
	}
	return lab
}

func generateCells(gridSize int) []HotZoneCell {
	n := gridSize * gridSize
	cells := make([]HotZoneCell, n)
	for i := range cells {
		cells[i] = HotZoneCell{
			Glyph:   glyphs[rand.IntN(len(glyphs))],
			Palette: rand.IntN(len(HotZonePalette)),
		}
	}
	return cells
}

// Settings returns a snapshot of the current settings.
func (l *HotZoneLab) Settings() HotZoneSettings {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.settings
}

// UpdateSettings applies changes under write lock.
func (l *HotZoneLab) UpdateSettings(fn func(s *HotZoneSettings)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fn(&l.settings)
}

// Paused returns whether the simulator is paused.
func (l *HotZoneLab) Paused() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.paused
}

// TogglePause flips pause state and returns the new value.
func (l *HotZoneLab) TogglePause() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.paused = !l.paused
	return l.paused
}

// Region returns a snapshot of a single region (1-indexed).
func (l *HotZoneLab) Region(id int) HotZoneRegion {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if id < 1 || id > 64 {
		return HotZoneRegion{}
	}
	return l.regions[id-1]
}

// Regions returns snapshots of the active regions.
func (l *HotZoneLab) Regions() []HotZoneRegion {
	l.mu.RLock()
	defer l.mu.RUnlock()
	n := l.settings.RegionCount
	out := make([]HotZoneRegion, n)
	copy(out, l.regions[:n])
	return out
}

// ToggleLock flips the lock state of a region and returns the new state.
func (l *HotZoneLab) ToggleLock(id int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if id < 1 || id > 64 {
		return false
	}
	l.regions[id-1].Locked = !l.regions[id-1].Locked
	return l.regions[id-1].Locked
}

// SimTick runs one simulation tick.
func (l *HotZoneLab) SimTick() []int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UTC()
	n := l.settings.RegionCount
	gs := l.settings.GridSize
	var updated []int

	if l.settings.BurstMode {
		for i := 0; i < n; i++ {
			if l.regions[i].Locked {
				continue
			}
			l.regions[i].Counter++
			l.regions[i].LastUpdate = now
			l.regions[i].Cells = generateCells(gs)
			updated = append(updated, i+1)
		}
	} else if l.settings.FocusedRegion > 0 && l.settings.FocusedRegion <= n {
		idx := l.settings.FocusedRegion - 1
		if !l.regions[idx].Locked {
			l.regions[idx].Counter++
			l.regions[idx].LastUpdate = now
			l.regions[idx].Cells = generateCells(gs)
			updated = append(updated, idx+1)
		}
	} else {
		idx := rand.IntN(n)
		if !l.regions[idx].Locked {
			l.regions[idx].Counter++
			l.regions[idx].LastUpdate = now
			l.regions[idx].Cells = generateCells(gs)
			updated = append(updated, idx+1)
		}
	}

	return updated
}

// JitteredInterval returns the next tick duration with random jitter.
func (l *HotZoneLab) JitteredInterval() time.Duration {
	l.mu.RLock()
	defer l.mu.RUnlock()
	base := l.settings.UpdateIntervalMS
	jMin := l.settings.JitterMinMS
	jMax := l.settings.JitterMaxMS
	jitter := 0
	if jMax > jMin {
		jitter = jMin + rand.IntN(jMax-jMin+1)
	} else {
		jitter = jMin
	}
	return time.Duration(base+jitter) * time.Millisecond
}

// RecordActivity appends an entry, keeping the last 30.
func (l *HotZoneLab) RecordActivity(action string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.activity = append(l.activity, HotZoneActivity{
		Timestamp: time.Now().UTC(),
		Action:    action,
	})
	if len(l.activity) > 30 {
		l.activity = l.activity[len(l.activity)-30:]
	}
}

// Activity returns the recent activity log.
func (l *HotZoneLab) Activity() []HotZoneActivity {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]HotZoneActivity, len(l.activity))
	copy(out, l.activity)
	return out
}

// RecordReceived records that the server endpoint handled a command.
func (l *HotZoneLab) RecordReceived(mode HotZoneMode) {
	switch mode {
	case HotZoneModeHXPost:
		l.hxReceived.Add(1)
	case HotZoneModeTavern:
		l.tavernReceived.Add(1)
	}
}

// RecordLifecycle records a client-reported command lifecycle event.
func (l *HotZoneLab) RecordLifecycle(mode HotZoneMode, action string) {
	switch mode {
	case HotZoneModeHXPost:
		switch action {
		case "dispatched":
			l.hxDispatched.Add(1)
		case "succeeded":
			l.hxOK.Add(1)
		case "failed":
			l.hxFail.Add(1)
		}
	case HotZoneModeTavern:
		switch action {
		case "dispatched":
			l.tavernDispatched.Add(1)
		case "succeeded":
			l.tavernOK.Add(1)
		case "failed":
			l.tavernFail.Add(1)
		}
	}
}

// CommandStats returns delivery stats for both modes.
func (l *HotZoneLab) CommandStats() [2]HotZoneCommandStat {
	return [2]HotZoneCommandStat{
		{Mode: HotZoneModeHXPost, Dispatched: l.hxDispatched.Load(), Received: l.hxReceived.Load(), Succeeded: l.hxOK.Load(), Failed: l.hxFail.Load()},
		{Mode: HotZoneModeTavern, Dispatched: l.tavernDispatched.Load(), Received: l.tavernReceived.Load(), Succeeded: l.tavernOK.Load(), Failed: l.tavernFail.Load()},
	}
}

// ResetStats zeroes all command counters.
func (l *HotZoneLab) ResetStats() {
	l.hxDispatched.Store(0)
	l.hxReceived.Store(0)
	l.hxOK.Store(0)
	l.hxFail.Store(0)
	l.tavernDispatched.Store(0)
	l.tavernReceived.Store(0)
	l.tavernOK.Store(0)
	l.tavernFail.Store(0)
}
