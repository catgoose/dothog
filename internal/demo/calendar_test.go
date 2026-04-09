// setup:feature:demo

package demo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalendarStore_AddAndRemoveEvent(t *testing.T) {
	s := NewCalendarStore()
	day := time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)

	id := s.AddEvent(day, "Test event", "Alice", CalCatReminder)
	assert.NotZero(t, id)

	events := s.EventsForDay(day)
	require.NotEmpty(t, events)

	var found bool
	for _, e := range events {
		if e.ID == id {
			found = true
			assert.Equal(t, "Test event", e.Title)
			assert.Equal(t, "Alice", e.Assignee)
			assert.Equal(t, CalCatReminder, e.Category)
			break
		}
	}
	assert.True(t, found, "added event should appear in EventsForDay")

	removed := s.RemoveEvent(id)
	assert.True(t, removed)

	events = s.EventsForDay(day)
	for _, e := range events {
		assert.NotEqual(t, id, e.ID, "removed event should not appear")
	}
}

func TestCalendarStore_RemoveUnknownReturnsFalse(t *testing.T) {
	s := NewCalendarStore()
	assert.False(t, s.RemoveEvent(999999))
}

func TestCalendarStore_EventsForMonth(t *testing.T) {
	s := NewCalendarStore()
	year, month := 2026, time.July
	day := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)

	s.AddEvent(day, "First", "", CalCatMaintenance)
	s.AddEvent(day.AddDate(0, 0, 14), "Mid", "", CalCatAppointment)
	s.AddEvent(day.AddDate(0, 1, 0), "NextMonth", "", CalCatDeadline)

	events := s.EventsForMonth(year, month)
	require.GreaterOrEqual(t, len(events), 2)

	// All returned events should be in the requested month.
	for _, e := range events {
		assert.Equal(t, year, e.Date.Year())
		assert.Equal(t, month, e.Date.Month())
	}
}

func TestCalendarStore_DayCountsForMonth(t *testing.T) {
	s := NewCalendarStore()
	year, month := 2026, time.July

	s.AddEvent(time.Date(year, month, 5, 0, 0, 0, 0, time.UTC), "A", "", CalCatMaintenance)
	s.AddEvent(time.Date(year, month, 5, 0, 0, 0, 0, time.UTC), "B", "", CalCatAppointment)
	s.AddEvent(time.Date(year, month, 5, 0, 0, 0, 0, time.UTC), "C", "", CalCatMaintenance)
	s.AddEvent(time.Date(year, month, 10, 0, 0, 0, 0, time.UTC), "D", "", CalCatReminder)

	counts := s.DayCountsForMonth(year, month)
	assert.Equal(t, 2, counts[5][CalCatMaintenance])
	assert.Equal(t, 1, counts[5][CalCatAppointment])
	assert.Equal(t, 1, counts[10][CalCatReminder])
}

func TestCalendarStore_EventsForDayDifferentTimesSameDay(t *testing.T) {
	s := NewCalendarStore()
	morning := time.Date(2026, time.July, 15, 9, 30, 0, 0, time.UTC)
	evening := time.Date(2026, time.July, 15, 21, 0, 0, 0, time.UTC)

	s.AddEvent(morning, "Morning", "", CalCatAppointment)
	s.AddEvent(evening, "Evening", "", CalCatReminder)

	// Both should be returned regardless of the time-of-day used to query.
	events := s.EventsForDay(time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC))
	assert.Len(t, events, 2)
}

func TestCalendarStore_SeededWithCurrentMonth(t *testing.T) {
	s := NewCalendarStore()
	now := time.Now().UTC()
	events := s.EventsForMonth(now.Year(), now.Month())
	assert.NotEmpty(t, events, "store should be seeded with sample events for the current month")
}
