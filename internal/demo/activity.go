// setup:feature:demo

package demo

import (
	"sync"
	"time"
)

// ActivityEvent is a single audit entry; ID is monotonic per log and used by Since for incremental polling.
type ActivityEvent struct {
	Timestamp  time.Time
	Action     string
	Resource   string
	Name       string
	Detail     string
	ID         int
	ResourceID int
}

// ActivityLog is a thread-safe capped event log.
type ActivityLog struct {
	events []ActivityEvent
	nextID int
	maxLen int
	mu     sync.RWMutex
}

// NewActivityLog caps retention at maxLen events; older entries are dropped on Record.
func NewActivityLog(maxLen int) *ActivityLog {
	return &ActivityLog{maxLen: maxLen}
}

// Record appends an event with a fresh monotonic ID, evicting the oldest when the cap is exceeded.
func (l *ActivityLog) Record(action, resource string, resourceID int, name, detail string) ActivityEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.nextID++
	e := ActivityEvent{
		ID:         l.nextID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Name:       name,
		Detail:     detail,
		Timestamp:  time.Now(),
	}
	l.events = append(l.events, e)
	if len(l.events) > l.maxLen {
		l.events = l.events[len(l.events)-l.maxLen:]
	}
	return e
}

// Recent yields up to the last n events, newest first; n is clamped to the log size.
func (l *ActivityLog) Recent(n int) []ActivityEvent {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if n > len(l.events) {
		n = len(l.events)
	}
	result := make([]ActivityEvent, n)
	for i := 0; i < n; i++ {
		result[i] = l.events[len(l.events)-1-i]
	}
	return result
}

// Since yields events with ID > afterID in insertion order; designed for incremental polling.
func (l *ActivityLog) Since(afterID int) []ActivityEvent {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var result []ActivityEvent
	for _, e := range l.events {
		if e.ID > afterID {
			result = append(result, e)
		}
	}
	return result
}

// Len is the current event count (bounded by maxLen).
func (l *ActivityLog) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.events)
}
