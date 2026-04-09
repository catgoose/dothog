// setup:feature:demo

package demo

import "sync/atomic"

// PublishStats is a small atomic counter pair used by Tavern demo labs to
// track how many publishes have flowed through their topics and how many
// bytes those publishes carried. Both the document and hooks demos surface
// this in their UI; sharing the type keeps the shape consistent across
// labs and lets either demo embed it without re-implementing the same two
// fields.
type PublishStats struct {
	count atomic.Int64
	bytes atomic.Int64
}

// Add records a single publish carrying the given byte count.
func (s *PublishStats) Add(bytes int) {
	s.count.Add(1)
	s.bytes.Add(int64(bytes))
}

// Snapshot returns the current count and total byte total.
func (s *PublishStats) Snapshot() (count, bytes int64) {
	return s.count.Load(), s.bytes.Load()
}

// Reset zeros both counters. Used by demos that expose a "reset" control.
func (s *PublishStats) Reset() {
	s.count.Store(0)
	s.bytes.Store(0)
}
