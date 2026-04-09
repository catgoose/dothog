// setup:feature:demo

package demo

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// CalendarEventCategory groups events by domain so the UI can color-code
// and filter them. The names are deliberately generic so the same demo can
// be reused for maintenance schedules, appointments, classes, and so on.
type CalendarEventCategory string

// Calendar event category constants.
const (
	CalCatMaintenance CalendarEventCategory = "maintenance"
	CalCatAppointment CalendarEventCategory = "appointment"
	CalCatReminder    CalendarEventCategory = "reminder"
	CalCatDeadline    CalendarEventCategory = "deadline"
)

// AllCalendarCategories lists every category in display order.
var AllCalendarCategories = []CalendarEventCategory{
	CalCatMaintenance, CalCatAppointment, CalCatReminder, CalCatDeadline,
}

// CalendarEvent is a single scheduled item.
type CalendarEvent struct {
	Date     time.Time
	Title    string
	Assignee string
	Category CalendarEventCategory
	ID       int64
}

// CalendarStore is an in-memory store of demo calendar events.
type CalendarStore struct {
	events  []CalendarEvent
	counter atomic.Int64
	mu      sync.RWMutex
}

// NewCalendarStore creates a new store seeded with sample events for the
// current month so the demo always has something to show.
func NewCalendarStore() *CalendarStore {
	s := &CalendarStore{}
	s.seedCurrentMonth()
	return s
}

// AddEvent adds an event and returns the assigned ID.
func (s *CalendarStore) AddEvent(date time.Time, title, assignee string, cat CalendarEventCategory) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.counter.Add(1)
	s.events = append(s.events, CalendarEvent{
		ID:       id,
		Date:     truncateToDay(date),
		Title:    title,
		Category: cat,
		Assignee: assignee,
	})
	return id
}

// RemoveEvent removes an event by ID. Returns true if the event existed.
func (s *CalendarStore) RemoveEvent(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, e := range s.events {
		if e.ID == id {
			s.events = append(s.events[:i], s.events[i+1:]...)
			return true
		}
	}
	return false
}

// EventsForMonth returns all events whose Date falls in the given year/month,
// sorted by date then by ID for stable display order.
func (s *CalendarStore) EventsForMonth(year int, month time.Month) []CalendarEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []CalendarEvent
	for _, e := range s.events {
		if e.Date.Year() == year && e.Date.Month() == month {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date.Equal(out[j].Date) {
			return out[i].ID < out[j].ID
		}
		return out[i].Date.Before(out[j].Date)
	})
	return out
}

// EventsForDay returns all events on the given day.
func (s *CalendarStore) EventsForDay(day time.Time) []CalendarEvent {
	target := truncateToDay(day)
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []CalendarEvent
	for _, e := range s.events {
		if e.Date.Equal(target) {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// DayCounts is a per-day map of category to event count, used by the
// month-grid renderer to draw indicator dots.
type DayCounts map[CalendarEventCategory]int

// DayCountsForMonth returns a map keyed by day-of-month with category counts.
func (s *CalendarStore) DayCountsForMonth(year int, month time.Month) map[int]DayCounts {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[int]DayCounts)
	for _, e := range s.events {
		if e.Date.Year() != year || e.Date.Month() != month {
			continue
		}
		day := e.Date.Day()
		if out[day] == nil {
			out[day] = make(DayCounts)
		}
		out[day][e.Category]++
	}
	return out
}

// truncateToDay returns the date with hour/min/sec/ns zeroed (UTC).
func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// seedCurrentMonth populates a few sample events for the current month so the
// demo isn't empty on first load.
func (s *CalendarStore) seedCurrentMonth() {
	now := time.Now().UTC()
	year, month := now.Year(), now.Month()
	day := func(d int) time.Time {
		return time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
	}
	samples := []struct {
		title    string
		assignee string
		cat      CalendarEventCategory
		dom      int
	}{
		{"HVAC filter swap", "Jordan", CalCatMaintenance, 3},
		{"Quarterly inspection", "Maria", CalCatMaintenance, 12},
		{"Standup with vendor", "Sam", CalCatAppointment, 5},
		{"Renew SSL cert", "ops-team", CalCatDeadline, 18},
		{"Backup tape rotation", "Pat", CalCatReminder, 8},
		{"Pump pressure test", "Jordan", CalCatMaintenance, 22},
		{"Quote approval due", "Maria", CalCatDeadline, 14},
		{"Followup with client", "Sam", CalCatAppointment, 25},
	}
	for _, x := range samples {
		id := s.counter.Add(1)
		s.events = append(s.events, CalendarEvent{
			ID:       id,
			Date:     day(x.dom),
			Title:    x.title,
			Category: x.cat,
			Assignee: x.assignee,
		})
	}
}
