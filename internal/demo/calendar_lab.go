// setup:feature:demo

package demo

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

// CalendarLabSettings holds the server-authoritative control state for the
// calendar lab. All fields are safe to read/write under the parent CalendarLab
// mutex.
type CalendarLabSettings struct {
	VisibleCategories map[CalendarEventCategory]bool
	Assignee          string // "" = all
	Density           int    // max events shown per day cell (1–8)
	SimSpeed          int    // ms between ticks (100–5000)
	BurstSize         int    // synthetic events per tick (1–5)
	CompactMode       bool
	HighlightWeekends bool
}

// CalendarLabActivity records one simulator or user action for the activity log.
type CalendarLabActivity struct {
	Timestamp time.Time
	Action    string
}

// CalendarLab wraps a CalendarStore with shared demo state: view settings,
// simulator counters, selected day, and an activity log. All methods are
// goroutine-safe.
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

// NewCalendarLab creates a new lab backed by a fresh CalendarStore, initialised
// to the current month with default settings.
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
			Density:           4,
			SimSpeed:          2000,
			BurstSize:         1,
			VisibleCategories: cats,
		},
	}
}

// Year returns the current visible year.
func (l *CalendarLab) Year() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.year
}

// Month returns the current visible month.
func (l *CalendarLab) Month() time.Month {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.month
}

// SetMonth changes the visible month/year.
func (l *CalendarLab) SetMonth(year int, month time.Month) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.year = year
	l.month = month
}

// SelectedDay returns the currently selected day (zero if none).
func (l *CalendarLab) SelectedDay() time.Time {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.selectedDay
}

// SelectDay sets the currently inspected day.
func (l *CalendarLab) SelectDay(day time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.selectedDay = truncateToDay(day)
}

// Settings returns a snapshot of the current control state.
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

// Paused returns whether the simulator is paused.
func (l *CalendarLab) Paused() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.paused
}

// TogglePause flips the paused state and returns the new value.
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

// Activity returns the recent activity log (newest last).
func (l *CalendarLab) Activity() []CalendarLabActivity {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]CalendarLabActivity, len(l.activity))
	copy(out, l.activity)
	return out
}

// SimTick runs one simulator tick: adds BurstSize random events to the
// current month, prunes back to a budget, and returns descriptions of
// what happened. The caller is responsible for publishing.
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
		title := fmt.Sprintf("Auto: %s %s", verbs[rand.IntN(len(verbs))], string(cat))
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		l.Store.AddEvent(date, title, assignee, cat)
		actions = append(actions, fmt.Sprintf("%s on %s (%s)", title, date.Format("Jan 2"), assignee))
	}

	// Prune: keep at most 60 events in the visible month.
	const budget = 60
	events := l.Store.EventsForMonth(year, month)
	if len(events) > budget {
		excess := len(events) - budget
		for i := 0; i < excess; i++ {
			l.Store.RemoveEvent(events[i].ID)
		}
		actions = append(actions, fmt.Sprintf("pruned %d oldest events", excess))
	}

	return actions
}

// EventCount returns the total number of events in the current visible month.
func (l *CalendarLab) EventCount() int {
	l.mu.RLock()
	year, month := l.year, l.month
	l.mu.RUnlock()
	return len(l.Store.EventsForMonth(year, month))
}
