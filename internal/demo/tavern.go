// setup:feature:demo

package demo

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ReplayLab tracks numbered events for the replay demo.
type ReplayLab struct {
	counter      atomic.Int64
	replayWindow atomic.Int64
}

// NewReplayLab creates a new replay lab with the given initial window.
func NewReplayLab(window int) *ReplayLab {
	rl := &ReplayLab{}
	rl.replayWindow.Store(int64(window))
	return rl
}

// NextEvent returns the next event ID and sequence number.
func (rl *ReplayLab) NextEvent() (id string, seq int64) {
	n := rl.counter.Add(1)
	return fmt.Sprintf("replay-%d", n), n
}

// ReplayWindow returns the current replay window size.
func (rl *ReplayLab) ReplayWindow() int {
	return int(rl.replayWindow.Load())
}

// SetReplayWindow updates the replay window size.
func (rl *ReplayLab) SetReplayWindow(n int) {
	rl.replayWindow.Store(int64(n))
}

// Reset zeroes the sequence counter so the next event starts at replay-1.
func (rl *ReplayLab) Reset() {
	rl.counter.Store(0)
}

// BackpressureLab tracks stress presets and tier changes.
type BackpressureLab struct {
	currentTiers map[string]liveTier
	activePreset string
	tierChanges  []TierChange
	mu           sync.RWMutex
}

type liveTier struct {
	updated time.Time
	tier    int
}

// TierChange records a backpressure tier transition.
type TierChange struct {
	Timestamp time.Time
	Topic     string
	SubID     string
	OldTier   int
	NewTier   int
}

// NewBackpressureLab creates a new backpressure lab in calm state.
func NewBackpressureLab() *BackpressureLab {
	return &BackpressureLab{
		activePreset: "calm",
		currentTiers: make(map[string]liveTier),
	}
}

// BackpressurePresets maps preset names to publish intervals.
var BackpressurePresets = map[string]time.Duration{
	"calm":         2 * time.Second,
	"moderate":     200 * time.Millisecond,
	"heavy":        50 * time.Millisecond,
	"overwhelming": 5 * time.Millisecond,
}

// ActivePreset returns the current stress preset name.
func (bl *BackpressureLab) ActivePreset() string {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	return bl.activePreset
}

// SetPreset changes the active stress preset.
func (bl *BackpressureLab) SetPreset(name string) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	if _, ok := BackpressurePresets[name]; ok {
		bl.activePreset = name
	}
}

// RecordTierChange logs a backpressure tier transition and updates live state.
func (bl *BackpressureLab) RecordTierChange(topic, subID string, oldTier, newTier int) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.tierChanges = append(bl.tierChanges, TierChange{
		Timestamp: time.Now(),
		Topic:     topic,
		SubID:     subID,
		OldTier:   oldTier,
		NewTier:   newTier,
	})
	// Keep last 30 entries.
	if len(bl.tierChanges) > 30 {
		bl.tierChanges = bl.tierChanges[len(bl.tierChanges)-30:]
	}
	key := topic + "/" + subID
	if newTier == 3 { // disconnect = evicted
		delete(bl.currentTiers, key)
	} else {
		bl.currentTiers[key] = liveTier{tier: newTier, updated: time.Now()}
	}
}

// HighestTier returns the highest tier among live subscribers, or 0 (normal)
// if no subscribers are in an elevated tier. Entries older than 10s are
// treated as stale (subscriber left without a tier-change callback).
func (bl *BackpressureLab) HighestTier() int {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	highest := 0
	now := time.Now()
	for key, lt := range bl.currentTiers {
		if now.Sub(lt.updated) > 10*time.Second {
			delete(bl.currentTiers, key)
			continue
		}
		if lt.tier > highest {
			highest = lt.tier
		}
	}
	return highest
}

// TierChanges returns a copy of recent tier change events.
func (bl *BackpressureLab) TierChanges() []TierChange {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	out := make([]TierChange, len(bl.tierChanges))
	copy(out, bl.tierChanges)
	return out
}

// PublishLab tracks publish counts for side-by-side comparison.
type PublishLab struct {
	RawCount       atomic.Int64
	DebouncedCount atomic.Int64
	ThrottledCount atomic.Int64
	IfChangedCount atomic.Int64
}

// NewPublishLab creates a new publish lab.
func NewPublishLab() *PublishLab {
	return &PublishLab{}
}

// Reset zeroes all counters.
func (pl *PublishLab) Reset() {
	pl.RawCount.Store(0)
	pl.DebouncedCount.Store(0)
	pl.ThrottledCount.Store(0)
	pl.IfChangedCount.Store(0)
}

// HooksLab tracks mutation source and hook events.
type HooksLab struct {
	source   string
	hookLog  []HookEvent
	pubCount atomic.Int64
	pubBytes atomic.Int64
	mu       sync.RWMutex
}

// HookEvent records a hook firing.
type HookEvent struct {
	Timestamp time.Time
	HookType  string // "before", "after", "on-mutate"
	Detail    string
}

// NewHooksLab creates a new hooks lab.
func NewHooksLab() *HooksLab {
	return &HooksLab{source: "Hello from the Hooks Lab!"}
}

// Source returns the current source text.
func (hl *HooksLab) Source() string {
	hl.mu.RLock()
	defer hl.mu.RUnlock()
	return hl.source
}

// Update sets the source text.
func (hl *HooksLab) Update(text string) {
	hl.mu.Lock()
	defer hl.mu.Unlock()
	hl.source = text
}

// RecordHook logs a hook event.
func (hl *HooksLab) RecordHook(hookType, detail string) {
	hl.mu.Lock()
	defer hl.mu.Unlock()
	hl.hookLog = append(hl.hookLog, HookEvent{
		Timestamp: time.Now(),
		HookType:  hookType,
		Detail:    detail,
	})
	if len(hl.hookLog) > 20 {
		hl.hookLog = hl.hookLog[len(hl.hookLog)-20:]
	}
}

// HookLog returns a copy of recent hook events.
func (hl *HooksLab) HookLog() []HookEvent {
	hl.mu.RLock()
	defer hl.mu.RUnlock()
	out := make([]HookEvent, len(hl.hookLog))
	copy(out, hl.hookLog)
	return out
}

// AddPublishStats increments middleware counters.
func (hl *HooksLab) AddPublishStats(bytes int) {
	hl.pubCount.Add(1)
	hl.pubBytes.Add(int64(bytes))
}

// PublishStats returns total publish count and bytes.
func (hl *HooksLab) PublishStats() (count, bytes int64) {
	return hl.pubCount.Load(), hl.pubBytes.Load()
}
