// Package requestlog provides an in-memory ring buffer that captures recent
// slog records per request ID so they can be retrieved for debugging.
package requestlog

import (
	"sync"
	"time"
)

// Entry is a single captured log record.
type Entry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"msg"`
	Attrs   string    `json:"attrs,omitempty"`
}

// requestBucket holds all captured entries for one request.
type requestBucket struct {
	entries []Entry
	created time.Time
}

// Store is a bounded, thread-safe ring buffer of request log entries.
// When the maximum number of tracked requests is exceeded the oldest
// request's entries are evicted.
type Store struct {
	mu      sync.RWMutex
	buckets map[string]*requestBucket
	order   []string // insertion order for eviction
	maxReqs int
}

// NewStore creates a Store that retains logs for up to maxRequests recent requests.
func NewStore(maxRequests int) *Store {
	if maxRequests <= 0 {
		maxRequests = 500
	}
	return &Store{
		buckets: make(map[string]*requestBucket, maxRequests),
		order:   make([]string, 0, maxRequests),
		maxReqs: maxRequests,
	}
}

// Append adds a log entry for the given request ID.
func (s *Store) Append(requestID string, e Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.buckets[requestID]
	if !ok {
		// Evict oldest if at capacity.
		if len(s.order) >= s.maxReqs {
			oldest := s.order[0]
			s.order = s.order[1:]
			delete(s.buckets, oldest)
		}
		b = &requestBucket{created: time.Now()}
		s.buckets[requestID] = b
		s.order = append(s.order, requestID)
	}
	b.entries = append(b.entries, e)
}

// Get returns all captured entries for a request ID, or nil if not found.
func (s *Store) Get(requestID string) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, ok := s.buckets[requestID]
	if !ok {
		return nil
	}
	out := make([]Entry, len(b.entries))
	copy(out, b.entries)
	return out
}

