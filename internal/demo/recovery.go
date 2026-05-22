// setup:feature:demo

package demo

import (
	"sync"
	"sync/atomic"
	"time"
)

// RecoveryLab tracks state for the multiplexed reconnect/recovery demo.
// Three regions on the demo page each illustrate a different recovery
// strategy when a single SSE connection is dropped and reconnected:
//
//   - Replay region:  events flow into a Last-Event-ID replay buffer, so
//     reconnecting clients receive any missed events in order.
//   - Snapshot region: a single current value updated in place; subscribers
//     get the latest value on connect via SubscribeWithSnapshot, with no
//     event history.
//   - Live region:    pure live stream with no replay and no snapshot;
//     subscribers see only events that arrive after they connect.
type RecoveryLab struct {
	snapshotValue string
	replaySeq     atomic.Int64
	liveSeq       atomic.Int64
	mu            sync.RWMutex
}

// NewRecoveryLab pre-seeds the snapshot with "initial value" and both counters at 0.
func NewRecoveryLab() *RecoveryLab {
	return &RecoveryLab{snapshotValue: "initial value"}
}

// NextReplayEvent emits a fresh "rep-N" ID plus seq for PublishWithID, so
// the broker's replay cache picks the event up.
func (rl *RecoveryLab) NextReplayEvent() (id string, seq int64, ts time.Time) {
	n := rl.replaySeq.Add(1)
	return formatRecoveryID("rep", n), n, time.Now()
}

// NextLiveEvent advances the live-only counter; no replay ID is emitted.
func (rl *RecoveryLab) NextLiveEvent() (seq int64, ts time.Time) {
	return rl.liveSeq.Add(1), time.Now()
}

// SetSnapshot replaces the current snapshot value broadcast to new subscribers on connect.
func (rl *RecoveryLab) SetSnapshot(v string) {
	rl.mu.Lock()
	rl.snapshotValue = v
	rl.mu.Unlock()
}

// Snapshot is safe for concurrent reads alongside SetSnapshot writers.
func (rl *RecoveryLab) Snapshot() string {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.snapshotValue
}

// Reset zeros all sequence counters and resets the snapshot value. Used by
// the page's reset control so the demo can start fresh.
func (rl *RecoveryLab) Reset() {
	rl.replaySeq.Store(0)
	rl.liveSeq.Store(0)
	rl.mu.Lock()
	rl.snapshotValue = "initial value"
	rl.mu.Unlock()
}

// formatRecoveryID returns a stable event ID for the given prefix and seq.
func formatRecoveryID(prefix string, seq int64) string {
	return prefix + "-" + itoa(seq)
}

// itoa is a tiny int64 → string helper to avoid pulling fmt in for one call.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
