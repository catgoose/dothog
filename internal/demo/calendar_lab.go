// setup:feature:demo

package demo

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"sync"
	"time"
)

// CalendarLabSettings holds the server-authoritative control state for the
// calendar lab. All fields are safe to read/write under the parent CalendarLab
// mutex.
type CalendarLabSettings struct {
	VisibleCategories map[CalendarEventCategory]bool
	Assignee          string // "" = all
	Density           int    // max events shown per day cell (1–12)
	SimSpeed          int    // ms between ticks (10–5000)
	BurstSize         int    // synthetic events per tick (1–8)
	CompactMode       bool
	HighlightWeekends bool
}

// CalendarLabPreset names a density/speed combo selectable from the lab toolbar.
type CalendarLabPreset struct {
	Name      string
	Density   int
	SimSpeed  int
	BurstSize int
}

// CalendarLabPresets enumerates the lab's stress presets in ascending intensity order.
var CalendarLabPresets = []CalendarLabPreset{
	{Name: "Calm", Density: 2, SimSpeed: 4000, BurstSize: 1},
	{Name: "Steady", Density: 4, SimSpeed: 2000, BurstSize: 2},
	{Name: "Busy", Density: 6, SimSpeed: 800, BurstSize: 3},
	{Name: "Chaos", Density: 8, SimSpeed: 200, BurstSize: 5},
	{Name: "Hell", Density: 12, SimSpeed: 50, BurstSize: 8},
}

// CalendarLabActivity is a single entry in the lab's rolling 50-event activity feed.
type CalendarLabActivity struct {
	Timestamp time.Time
	Action    string
}

// CalendarLab is a CalendarStore plus shared demo state: view settings,
// simulator counters, selected day, and activity log. Methods are goroutine-safe.
type CalendarLab struct {
	selectedDay time.Time
	Store       *CalendarStore
	activity    []CalendarLabActivity
	settings    CalendarLabSettings
	year        int
	month       time.Month
	mu          sync.RWMutex
	paused      bool
}

// NewCalendarLab opens on the current UTC month with all categories visible and density=8.
func NewCalendarLab() *CalendarLab {
	now := time.Now().UTC()
	cats := make(map[CalendarEventCategory]bool, len(AllCalendarCategories))
	for _, c := range AllCalendarCategories {
		cats[c] = true
	}
	return &CalendarLab{
		Store: NewCalendarStore(),
		year:  now.Year(),
		month: now.Month(),
		settings: CalendarLabSettings{
			Density:           8,
			SimSpeed:          200,
			BurstSize:         5,
			VisibleCategories: cats,
		},
	}
}

// Year is the visible year under read lock.
func (l *CalendarLab) Year() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.year
}

// Month is the visible month under read lock.
func (l *CalendarLab) Month() time.Month {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.month
}

// SetMonth changes the visible month/year and clears the selected day if it
// falls outside the new visible month.
func (l *CalendarLab) SetMonth(year int, month time.Month) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.year = year
	l.month = month
	// Clear selection if it falls outside the new visible month.
	if !l.selectedDay.IsZero() && (l.selectedDay.Year() != year || l.selectedDay.Month() != month) {
		l.selectedDay = time.Time{}
	}
}

// SelectedDay is the zero time when nothing is selected.
func (l *CalendarLab) SelectedDay() time.Time {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.selectedDay
}

// SelectDay truncates to UTC midnight; pass the zero time to clear the selection.
func (l *CalendarLab) SelectDay(day time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.selectedDay = truncateToDay(day)
}

// Settings is a deep-copied snapshot; the VisibleCategories map is safe to iterate.
func (l *CalendarLab) Settings() CalendarLabSettings {
	l.mu.RLock()
	defer l.mu.RUnlock()
	s := l.settings
	// Deep-copy the map so callers can iterate safely.
	cats := make(map[CalendarEventCategory]bool, len(l.settings.VisibleCategories))
	for k, v := range l.settings.VisibleCategories {
		cats[k] = v
	}
	s.VisibleCategories = cats
	return s
}

// UpdateSettings applies the given function under the write lock.
func (l *CalendarLab) UpdateSettings(fn func(s *CalendarLabSettings)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fn(&l.settings)
}

// Paused reports whether SimTick should be skipped this round.
func (l *CalendarLab) Paused() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.paused
}

// TogglePause flips the pause flag and returns the new value.
func (l *CalendarLab) TogglePause() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.paused = !l.paused
	return l.paused
}

// RecordActivity appends an entry to the activity log, keeping the last 50.
func (l *CalendarLab) RecordActivity(action string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.activity = append(l.activity, CalendarLabActivity{
		Timestamp: time.Now().UTC(),
		Action:    action,
	})
	if len(l.activity) > 50 {
		l.activity = l.activity[len(l.activity)-50:]
	}
}

// Activity is a defensive copy of the rolling 50-entry log, newest last.
func (l *CalendarLab) Activity() []CalendarLabActivity {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]CalendarLabActivity, len(l.activity))
	copy(out, l.activity)
	return out
}

// SimTick advances one step: adds BurstSize random events to the current month,
// prunes back to a 60-event budget by oldest ID, and returns human-readable actions; caller publishes.
func (l *CalendarLab) SimTick() []string {
	l.mu.RLock()
	year, month := l.year, l.month
	burst := l.settings.BurstSize
	l.mu.RUnlock()

	assignees := []string{"Jordan", "Maria", "Sam", "Pat", "ops-team"}
	verbs := []string{"scheduled", "moved", "added", "flagged"}

	var actions []string
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()

	for range burst {
		day := rand.IntN(daysInMonth) + 1
		cat := AllCalendarCategories[rand.IntN(len(AllCalendarCategories))]
		assignee := assignees[rand.IntN(len(assignees))]
		title := fmt.Sprintf("%s %s", verbs[rand.IntN(len(verbs))], string(cat))
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		l.Store.AddEvent(date, title, assignee, cat)
		actions = append(actions, fmt.Sprintf("%s on %s (%s)", title, date.Format("Jan 2"), assignee))
	}

	// Prune: keep at most 60 events in the visible month. Remove by
	// lowest ID (oldest insertion) rather than earliest date so the
	// surviving set stays spread across the whole month.
	const budget = 60
	events := l.Store.EventsForMonth(year, month)
	if len(events) > budget {
		// Build an ID-sorted slice to find the oldest insertions.
		byID := make([]CalendarEvent, len(events))
		copy(byID, events)
		sort.Slice(byID, func(i, j int) bool { return byID[i].ID < byID[j].ID })
		excess := len(byID) - budget
		for i := 0; i < excess; i++ {
			l.Store.RemoveEvent(byID[i].ID)
		}
		actions = append(actions, fmt.Sprintf("pruned %d oldest events", excess))
	}

	return actions
}

// EventCount totals every event in the visible month (no filters applied).
func (l *CalendarLab) EventCount() int {
	l.mu.RLock()
	year, month := l.year, l.month
	l.mu.RUnlock()
	return len(l.Store.EventsForMonth(year, month))
}

// FilteredEventCount counts visible-month events passing the active category and assignee filters.
func (l *CalendarLab) FilteredEventCount() int {
	l.mu.RLock()
	year, month := l.year, l.month
	settings := l.settings
	cats := make(map[CalendarEventCategory]bool, len(settings.VisibleCategories))
	for k, v := range settings.VisibleCategories {
		cats[k] = v
	}
	settings.VisibleCategories = cats
	l.mu.RUnlock()
	return len(FilterEvents(l.Store.EventsForMonth(year, month), settings))
}

// FilterEvents is pure: drops events failing the settings' category or assignee filters.
// The output reuses events' backing array, so callers must not retain both.
func FilterEvents(events []CalendarEvent, settings CalendarLabSettings) []CalendarEvent {
	out := events[:0:0]
	for _, e := range events {
		if settings.VisibleCategories[e.Category] && (settings.Assignee == "" || settings.Assignee == e.Assignee) {
			out = append(out, e)
		}
	}
	return out
}

// DayEventsMap groups visible-month events by day-of-month; the map is read-only.
func (l *CalendarLab) DayEventsMap() map[int][]CalendarEvent {
	l.mu.RLock()
	year, month := l.year, l.month
	l.mu.RUnlock()
	events := l.Store.EventsForMonth(year, month)
	m := make(map[int][]CalendarEvent)
	for _, e := range events {
		d := e.Date.Day()
		m[d] = append(m[d], e)
	}
	return m
}
