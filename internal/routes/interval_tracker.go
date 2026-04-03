// setup:feature:demo

package routes

import (
	"sync"
	"time"

	"github.com/catgoose/tavern"
)

// intervalTracker stores per-section durations and supports save/restore for
// master-control overrides.
type intervalTracker struct {
	mu    sync.RWMutex
	cur   map[string]time.Duration
	saved map[string]time.Duration
}

func newIntervalTracker(defaults map[string]time.Duration) *intervalTracker {
	cur := make(map[string]time.Duration, len(defaults))
	for k, v := range defaults {
		cur[k] = v
	}
	return &intervalTracker{cur: cur}
}

func (t *intervalTracker) set(pub *tavern.ScheduledPublisher, name string, d time.Duration) {
	t.mu.Lock()
	t.cur[name] = d
	t.mu.Unlock()
	pub.SetInterval(name, d)
}

func (t *intervalTracker) saveAndOverride(pub *tavern.ScheduledPublisher, d time.Duration) {
	t.mu.Lock()
	if t.saved == nil {
		t.saved = make(map[string]time.Duration, len(t.cur))
		for k, v := range t.cur {
			t.saved[k] = v
		}
	}
	for k := range t.cur {
		t.cur[k] = d
		pub.SetInterval(k, d)
	}
	t.mu.Unlock()
}

func (t *intervalTracker) restore(pub *tavern.ScheduledPublisher) {
	t.mu.Lock()
	if t.saved != nil {
		for k, v := range t.saved {
			t.cur[k] = v
			pub.SetInterval(k, v)
		}
		t.saved = nil
	}
	t.mu.Unlock()
}
