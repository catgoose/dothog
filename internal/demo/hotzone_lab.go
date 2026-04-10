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
	HotZoneModeHXPost  HotZoneMode = "hx-post"
	HotZoneModeTavern  HotZoneMode = "tavern-command"
)

// HotZoneRegion is a single independently-updating UI region.
type HotZoneRegion struct {
	ID           int
	Label        string
	Counter      int64 // number of SSE updates published
	LastUpdate   time.Time
	PayloadRunes int // how many filler runes per update
}

// HotZoneSettings holds operator-controlled simulation parameters.
type HotZoneSettings struct {
	UpdateIntervalMS int         // ms between ticks (50–5000)
	RegionCount      int         // how many regions to show (1–6)
	PayloadSize      int         // filler chars per region update (10–2000)
	BurstMode        bool        // burst: publish all regions every tick
	FocusedRegion    int         // 0 = random, 1–6 = only update that region
	CommandMode      HotZoneMode // which interaction pattern the UI uses
}

// HotZoneCommandStat tracks command delivery metrics per mode.
type HotZoneCommandStat struct {
	Mode      HotZoneMode
	Sent      int64
	Succeeded int64
	Failed    int64
}

// HotZoneLab wraps the shared state for the hot-zone stress surface.
type HotZoneLab struct {
	regions  [6]HotZoneRegion
	settings HotZoneSettings
	activity []HotZoneActivity
	mu       sync.RWMutex
	paused   bool

	hxSent      atomic.Int64
	hxOK        atomic.Int64
	hxFail      atomic.Int64
	tavernSent  atomic.Int64
	tavernOK    atomic.Int64
	tavernFail  atomic.Int64
}

// HotZoneActivity records one action in the activity log.
type HotZoneActivity struct {
	Timestamp time.Time
	Action    string
}

// NewHotZoneLab creates a lab with default settings.
func NewHotZoneLab() *HotZoneLab {
	lab := &HotZoneLab{
		settings: HotZoneSettings{
			UpdateIntervalMS: 500,
			RegionCount:      4,
			PayloadSize:      100,
			BurstMode:        false,
			FocusedRegion:    0,
			CommandMode:      HotZoneModeTavern,
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
	if id < 1 || id > 6 {
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

// RecordCommand records a command attempt for stats.
func (l *HotZoneLab) RecordCommand(mode HotZoneMode, success bool) {
	switch mode {
	case HotZoneModeHXPost:
		l.hxSent.Add(1)
		if success {
			l.hxOK.Add(1)
		} else {
			l.hxFail.Add(1)
		}
	case HotZoneModeTavern:
		l.tavernSent.Add(1)
		if success {
			l.tavernOK.Add(1)
		} else {
			l.tavernFail.Add(1)
		}
	}
}

// CommandStats returns delivery stats for both modes.
func (l *HotZoneLab) CommandStats() [2]HotZoneCommandStat {
	return [2]HotZoneCommandStat{
		{Mode: HotZoneModeHXPost, Sent: l.hxSent.Load(), Succeeded: l.hxOK.Load(), Failed: l.hxFail.Load()},
		{Mode: HotZoneModeTavern, Sent: l.tavernSent.Load(), Succeeded: l.tavernOK.Load(), Failed: l.tavernFail.Load()},
	}
}

// ResetStats zeroes all command counters.
func (l *HotZoneLab) ResetStats() {
	l.hxSent.Store(0)
	l.hxOK.Store(0)
	l.hxFail.Store(0)
	l.tavernSent.Store(0)
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
