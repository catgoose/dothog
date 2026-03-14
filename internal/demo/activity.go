// setup:feature:demo

package demo

import (
	"sync"
	"time"
)

// ActivityEvent represents a recorded action in the system.
type ActivityEvent struct {
	ID         int
	Action     string // created, updated, deleted, moved, approved, rejected, etc.
	Resource   string // person, task, request, contact, etc.
	ResourceID int
	Name       string // display name of the affected resource
	Detail     string
	Timestamp  time.Time
}

// ActivityLog is a thread-safe capped event log.
type ActivityLog struct {
	mu     sync.RWMutex
	events []ActivityEvent
	nextID int
	maxLen int
}

// NewActivityLog creates a log that retains at most maxLen events.
func NewActivityLog(maxLen int) *ActivityLog {
	return &ActivityLog{maxLen: maxLen}
}

// Record adds an event and returns it.
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

// Recent returns the last n events, newest first.
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

// Since returns all events after the given ID, oldest first.
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

// Len returns the current number of stored events.
func (l *ActivityLog) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.events)
}
