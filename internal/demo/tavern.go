// setup:feature:demo

package demo

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ReplayLab owns the monotonic event sequence and tunable replay window for the replay demo.
type ReplayLab struct {
	counter      atomic.Int64
	replayWindow atomic.Int64
}

// NewReplayLab starts with the counter at 0 and the given replay window.
func NewReplayLab(window int) *ReplayLab {
	rl := &ReplayLab{}
	rl.replayWindow.Store(int64(window))
	return rl
}

// NextEvent advances the counter and returns "replay-N" plus the new sequence number.
func (rl *ReplayLab) NextEvent() (id string, seq int64) {
	n := rl.counter.Add(1)
	return fmt.Sprintf("replay-%d", n), n
}

// ReplayWindow is the current size honoured by Last-Event-ID reconnects (atomic read).
func (rl *ReplayLab) ReplayWindow() int {
	return int(rl.replayWindow.Load())
}

// SetReplayWindow stores n atomically; takes effect on the next reconnect.
func (rl *ReplayLab) SetReplayWindow(n int) {
	rl.replayWindow.Store(int64(n))
}

// Reset zeroes the sequence counter so the next event starts at replay-1.
func (rl *ReplayLab) Reset() {
	rl.counter.Store(0)
}

// BackpressureLab tracks live subscriber tiers per (topic, subID) and a rolling 30-event tier-change log.
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

// NewBackpressureLab starts in the "calm" preset (2s publish interval).
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

// ActivePreset is the current stress-preset name under read lock.
func (bl *BackpressureLab) ActivePreset() string {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	return bl.activePreset
}

// SetPreset switches the publish interval; unknown names are silently ignored.
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

// HighestTier is the max tier across live subscribers (0 = normal); entries
// older than 10s are pruned as stale.
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

// TierChanges is a defensive copy of the rolling 30-entry tier-change log.
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

// NewPublishLab starts with all four counters at zero.
func NewPublishLab() *PublishLab {
	return &PublishLab{}
}

// Reset zeroes every raw/debounced/throttled/if-changed counter.
func (pl *PublishLab) Reset() {
	pl.RawCount.Store(0)
	pl.DebouncedCount.Store(0)
	pl.ThrottledCount.Store(0)
	pl.IfChangedCount.Store(0)
}

// HooksLab holds the editable source, a rolling 20-entry hook log, and publish-stats counters.
type HooksLab struct {
	source   string
	hookLog  []HookEvent
	pubStats PublishStats
	mu       sync.RWMutex
}

// HookEvent records one hook firing; HookType is one of "before", "after", "on-mutate".
type HookEvent struct {
	Timestamp time.Time
	HookType  string // "before", "after", "on-mutate"
	Detail    string
}

// NewHooksLab pre-populates Source with a starter greeting.
func NewHooksLab() *HooksLab {
	return &HooksLab{source: "Hello from the Hooks Lab!"}
}

// Source is safe for concurrent reads.
func (hl *HooksLab) Source() string {
	hl.mu.RLock()
	defer hl.mu.RUnlock()
	return hl.source
}

// Update replaces the editable source; the new value is broadcast to hooks on next publish.
func (hl *HooksLab) Update(text string) {
	hl.mu.Lock()
	defer hl.mu.Unlock()
	hl.source = text
}

// RecordHook appends a hook event, evicting the oldest once the 20-entry cap is exceeded.
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

// HookLog is a defensive copy of the rolling 20-entry hook log.
func (hl *HooksLab) HookLog() []HookEvent {
	hl.mu.RLock()
	defer hl.mu.RUnlock()
	out := make([]HookEvent, len(hl.hookLog))
	copy(out, hl.hookLog)
	return out
}

// AddPublishStats increments middleware counters.
func (hl *HooksLab) AddPublishStats(bytes int) {
	hl.pubStats.Add(bytes)
}

// PublishStats is the running count and byte total; the two atomic reads are not consistent.
func (hl *HooksLab) PublishStats() (count, bytes int64) {
	return hl.pubStats.Snapshot()
}
