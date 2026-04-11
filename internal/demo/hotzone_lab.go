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

// Command delivery modes for hot-zone regions.
const (
	HotZoneModeHXPost HotZoneMode = "hx-post"
	HotZoneModeTavern HotZoneMode = "tavern-command"
)

// HotZoneRegion is a single independently-updating UI region.
type HotZoneRegion struct {
	LastUpdate   time.Time
	Label        string
	Counter      int64 // number of SSE updates published
	ID           int
	PayloadRunes int // how many filler runes per update
}

// HotZoneSwapScope controls the granularity of SSE replacement.
type HotZoneSwapScope string

const (
	HotZoneSwapInner HotZoneSwapScope = "inner" // replace only the inner content (stable)
	HotZoneSwapCard  HotZoneSwapScope = "card"  // replace the entire card (fragile)
)

// HotZonePreset is a named pressure profile.
type HotZonePreset string

const (
	HotZonePresetNormal HotZonePreset = "normal"
	HotZonePresetHot    HotZonePreset = "hot"
	HotZonePresetNasty  HotZonePreset = "nasty"
	HotZonePresetHell   HotZonePreset = "hell"
)

// HotZoneSettings holds operator-controlled simulation parameters.
type HotZoneSettings struct {
	Preset           HotZonePreset    // active preset name
	CommandMode      HotZoneMode      // which interaction pattern the UI uses
	SwapScope        HotZoneSwapScope // how much of the region gets replaced
	UpdateIntervalMS int              // ms between ticks (25–5000)
	RegionCount      int              // how many regions to show (1–64)
	PayloadSize      int              // filler chars per region update (10–10240)
	FocusedRegion    int              // 0 = random, 1–8 = only update that region
	BurstMode        bool             // burst: publish all regions every tick
	// Heat-map visualization settings (client-side only).
	HeatEnabled    bool   // toggle heat gradient on/off
	HeatWindowMS   int    // rolling window for rate calculation (100–5000)
	HeatThreshold1 int    // updates/sec for first color stop
	HeatThreshold2 int    // updates/sec for second color stop
	HeatThreshold3 int    // updates/sec for third color stop
	HeatColor1     string // hex color at threshold 1 (e.g. "#22c55e")
	HeatColor2     string // hex color at threshold 2 (e.g. "#ef4444")
	HeatColor3     string // hex color at threshold 3 (e.g. "#a855f7")
	HeatBaseColor  string // hex color for idle/quiet state
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
		s.PayloadSize = 1000
		s.BurstMode = true
		s.FocusedRegion = 0
	case HotZonePresetNasty:
		s.UpdateIntervalMS = 75
		s.RegionCount = 16
		s.PayloadSize = 4000
		s.BurstMode = true
		s.FocusedRegion = 0
	case HotZonePresetHell:
		s.UpdateIntervalMS = 25
		s.RegionCount = 32
		s.PayloadSize = 10240
		s.BurstMode = true
		s.FocusedRegion = 0
	default: // normal
		s.UpdateIntervalMS = 500
		s.RegionCount = 4
		s.PayloadSize = 100
		s.BurstMode = false
		s.FocusedRegion = 0
	}
}

// HotZoneCommandStat tracks command lifecycle metrics per mode.
// Client-reported: Dispatched, Succeeded, Failed.
// Server-reported: Received.
type HotZoneCommandStat struct {
	Mode       HotZoneMode
	Dispatched int64
	Received   int64
	Succeeded  int64
	Failed     int64
}

// HotZoneLab wraps the shared state for the hot-zone stress surface.
type HotZoneLab struct {
	activity   []HotZoneActivity
	regions    [64]HotZoneRegion
	settings   HotZoneSettings
	hxDispatched     atomic.Int64
	hxReceived       atomic.Int64
	hxOK             atomic.Int64
	hxFail           atomic.Int64
	tavernDispatched atomic.Int64
	tavernReceived   atomic.Int64
	tavernOK         atomic.Int64
	tavernFail       atomic.Int64
	mu         sync.RWMutex
	paused     bool
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
			PayloadSize:      100,
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
	for i := range lab.regions {
		lab.regions[i] = HotZoneRegion{
			ID:           i + 1,
			Label:        fmt.Sprintf("Region %d", i+1),
			PayloadRunes: 100,
		}
	}
	return lab
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

// SimTick runs one simulation tick. Returns the IDs of regions that were
// updated so the caller can publish selectively.
func (l *HotZoneLab) SimTick() []int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UTC()
	n := l.settings.RegionCount
	var updated []int

	if l.settings.BurstMode {
		for i := 0; i < n; i++ {
			l.regions[i].Counter++
			l.regions[i].LastUpdate = now
			l.regions[i].PayloadRunes = l.settings.PayloadSize
			updated = append(updated, i+1)
		}
	} else if l.settings.FocusedRegion > 0 && l.settings.FocusedRegion <= n {
		idx := l.settings.FocusedRegion - 1
		l.regions[idx].Counter++
		l.regions[idx].LastUpdate = now
		l.regions[idx].PayloadRunes = l.settings.PayloadSize
		updated = append(updated, idx+1)
	} else {
		idx := rand.IntN(n)
		l.regions[idx].Counter++
		l.regions[idx].LastUpdate = now
		l.regions[idx].PayloadRunes = l.settings.PayloadSize
		updated = append(updated, idx+1)
	}

	return updated
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

// GeneratePayload creates a filler string of the given rune count.
func GeneratePayload(size int) string {
	if size <= 0 {
		return ""
	}
	words := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel"}
	var buf []byte
	for len(buf) < size {
		w := words[rand.IntN(len(words))]
		if len(buf) > 0 {
			buf = append(buf, ' ')
		}
		buf = append(buf, w...)
	}
	return string(buf[:size])
}
