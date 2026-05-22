// setup:feature:demo

package demo

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// maxTierChanges is the ring buffer capacity for tier change events.
const maxTierChanges = 50

// TierChangeEvent records a single backpressure tier transition.
type TierChangeEvent struct {
	Timestamp    time.Time
	Topic        string
	SubscriberID string
	TierName     string
	Tier         int
}

// ObservatoryState holds the observatory page's tier-change ring buffer, subscriber cap, and stress-test handle.
type ObservatoryState struct {
	stressCancel context.CancelFunc
	tierChanges  []TierChangeEvent
	mu           sync.RWMutex
	stressActive atomic.Bool
	maxPerTopic  atomic.Int32
}

// NewObservatoryState starts empty with the per-topic subscriber cap at 10.
func NewObservatoryState() *ObservatoryState {
	s := &ObservatoryState{
		tierChanges: make([]TierChangeEvent, 0, maxTierChanges),
	}
	s.maxPerTopic.Store(10)
	return s
}

// RecordTierChange appends a tier transition to the ring buffer, keeping the
// most recent maxTierChanges entries.
func (s *ObservatoryState) RecordTierChange(topic, subID string, tier int) {
	evt := TierChangeEvent{
		Timestamp:    time.Now(),
		Topic:        topic,
		SubscriberID: subID,
		Tier:         tier,
		TierName:     TierName(tier),
	}
	s.mu.Lock()
	s.tierChanges = append(s.tierChanges, evt)
	if len(s.tierChanges) > maxTierChanges {
		s.tierChanges = s.tierChanges[len(s.tierChanges)-maxTierChanges:]
	}
	s.mu.Unlock()
}

// RecentTierChanges is a copy of the ring buffer, reversed so the newest entry is index 0.
func (s *ObservatoryState) RecentTierChanges() []TierChangeEvent {
	s.mu.RLock()
	out := make([]TierChangeEvent, len(s.tierChanges))
	copy(out, s.tierChanges)
	s.mu.RUnlock()
	// Reverse so newest is first.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// MaxPerTopic is the current per-topic subscriber cap (atomic read).
func (s *ObservatoryState) MaxPerTopic() int {
	return int(s.maxPerTopic.Load())
}

// SetMaxPerTopic updates the cap; takes effect on the next admission check. Negative values clamp to 0.
func (s *ObservatoryState) SetMaxPerTopic(n int) {
	if n < 0 {
		n = 0
	}
	s.maxPerTopic.Store(int32(n))
}

// StressActive reports whether a stress-test goroutine is currently running.
func (s *ObservatoryState) StressActive() bool {
	return s.stressActive.Load()
}

// SetStress stores the cancel function for a running stress test and toggles the active flag.
func (s *ObservatoryState) SetStress(active bool, cancel context.CancelFunc) {
	s.mu.Lock()
	s.stressCancel = cancel
	s.mu.Unlock()
	s.stressActive.Store(active)
}

// CancelStress stops the running stress test if any.
func (s *ObservatoryState) CancelStress() {
	s.mu.Lock()
	if s.stressCancel != nil {
		s.stressCancel()
		s.stressCancel = nil
	}
	s.mu.Unlock()
	s.stressActive.Store(false)
}

// TierName maps an integer tier to its human-readable label.
func TierName(tier int) string {
	switch tier {
	case 0:
		return "Normal"
	case 1:
		return "Throttle"
	case 2:
		return "Simplify"
	case 3:
		return "Disconnect"
	default:
		return "Unknown"
	}
}
