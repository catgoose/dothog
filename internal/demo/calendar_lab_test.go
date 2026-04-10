// setup:feature:demo

package demo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalendarLab_NewLab(t *testing.T) {
	lab := NewCalendarLab()
	now := time.Now().UTC()

	assert.Equal(t, now.Year(), lab.Year())
	assert.Equal(t, now.Month(), lab.Month())
	assert.True(t, lab.SelectedDay().IsZero())
	assert.False(t, lab.Paused())
	assert.Greater(t, lab.EventCount(), 0, "should have seed events")

	settings := lab.Settings()
	assert.Equal(t, 4, settings.Density)
	assert.Equal(t, 2000, settings.SimSpeed)
	assert.Equal(t, 1, settings.BurstSize)
	for _, cat := range AllCalendarCategories {
		assert.True(t, settings.VisibleCategories[cat], "category %s should default to visible", cat)
	}
}

func TestCalendarLab_SelectDay(t *testing.T) {
	lab := NewCalendarLab()
	day := time.Date(2026, 4, 15, 12, 30, 0, 0, time.UTC)
	lab.SelectDay(day)

	selected := lab.SelectedDay()
	assert.Equal(t, 2026, selected.Year())
	assert.Equal(t, time.April, selected.Month())
	assert.Equal(t, 15, selected.Day())
	assert.Equal(t, 0, selected.Hour(), "should truncate to day")
}

func TestCalendarLab_SetMonth(t *testing.T) {
	lab := NewCalendarLab()
	lab.SetMonth(2025, time.December)
	assert.Equal(t, 2025, lab.Year())
	assert.Equal(t, time.December, lab.Month())
}

func TestCalendarLab_UpdateSettings(t *testing.T) {
	lab := NewCalendarLab()
	lab.UpdateSettings(func(s *CalendarLabSettings) {
		s.CompactMode = true
		s.Density = 2
		s.Assignee = "Jordan"
		s.VisibleCategories[CalCatMaintenance] = false
	})

	settings := lab.Settings()
	assert.True(t, settings.CompactMode)
	assert.Equal(t, 2, settings.Density)
	assert.Equal(t, "Jordan", settings.Assignee)
	assert.False(t, settings.VisibleCategories[CalCatMaintenance])
	assert.True(t, settings.VisibleCategories[CalCatAppointment])
}

func TestCalendarLab_TogglePause(t *testing.T) {
	lab := NewCalendarLab()
	assert.False(t, lab.Paused())

	paused := lab.TogglePause()
	assert.True(t, paused)
	assert.True(t, lab.Paused())

	paused = lab.TogglePause()
	assert.False(t, paused)
}

func TestCalendarLab_Activity(t *testing.T) {
	lab := NewCalendarLab()
	lab.RecordActivity("test action 1")
	lab.RecordActivity("test action 2")

	activity := lab.Activity()
	require.Len(t, activity, 2)
	assert.Equal(t, "test action 1", activity[0].Action)
	assert.Equal(t, "test action 2", activity[1].Action)
}

func TestCalendarLab_ActivityCap(t *testing.T) {
	lab := NewCalendarLab()
	for i := 0; i < 60; i++ {
		lab.RecordActivity("entry")
	}
	assert.Len(t, lab.Activity(), 50, "should cap at 50 entries")
}

func TestCalendarLab_SimTick(t *testing.T) {
	lab := NewCalendarLab()
	before := lab.EventCount()

	actions := lab.SimTick()
	assert.NotEmpty(t, actions)
	assert.GreaterOrEqual(t, lab.EventCount(), before)
}

func TestCalendarLab_SimTickBurst(t *testing.T) {
	lab := NewCalendarLab()
	lab.UpdateSettings(func(s *CalendarLabSettings) {
		s.BurstSize = 3
	})

	actions := lab.SimTick()
	// Should have at least 3 add actions (may also have a prune action)
	assert.GreaterOrEqual(t, len(actions), 3)
}

func TestCalendarLab_SettingsDeepCopy(t *testing.T) {
	lab := NewCalendarLab()
	s1 := lab.Settings()
	s1.VisibleCategories[CalCatMaintenance] = false

	s2 := lab.Settings()
	assert.True(t, s2.VisibleCategories[CalCatMaintenance],
		"mutating returned settings should not affect the lab")
}

func TestCalendarLab_SetMonth_ClearsSelection(t *testing.T) {
	lab := NewCalendarLab()

	// Select a day in April 2026.
	lab.SelectDay(time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC))
	require.Equal(t, 15, lab.SelectedDay().Day())

	// Navigate to May 2026 — selection should be cleared.
	lab.SetMonth(2026, time.May)
	assert.True(t, lab.SelectedDay().IsZero(),
		"SetMonth should clear selection when it falls outside the new month")
}

func TestCalendarLab_SetMonth_KeepsSelectionInSameMonth(t *testing.T) {
	lab := NewCalendarLab()
	now := time.Now().UTC()

	// Select a day in the current month.
	lab.SelectDay(time.Date(now.Year(), now.Month(), 5, 0, 0, 0, 0, time.UTC))
	require.False(t, lab.SelectedDay().IsZero())

	// Navigate to the same month — selection should be preserved.
	lab.SetMonth(now.Year(), now.Month())
	assert.False(t, lab.SelectedDay().IsZero(),
		"SetMonth should keep selection when it falls within the new month")
}

func TestCalendarLab_FilterEvents(t *testing.T) {
	events := []CalendarEvent{
		{ID: 1, Category: CalCatMaintenance, Assignee: "Jordan"},
		{ID: 2, Category: CalCatAppointment, Assignee: "Maria"},
		{ID: 3, Category: CalCatReminder, Assignee: "Jordan"},
		{ID: 4, Category: CalCatDeadline, Assignee: "Sam"},
	}

	cats := map[CalendarEventCategory]bool{
		CalCatMaintenance: true,
		CalCatAppointment: true,
		CalCatReminder:    true,
		CalCatDeadline:    true,
	}

	t.Run("all visible no assignee filter", func(t *testing.T) {
		settings := CalendarLabSettings{VisibleCategories: cats, Assignee: ""}
		got := FilterEvents(events, settings)
		assert.Len(t, got, 4)
	})

	t.Run("assignee filter", func(t *testing.T) {
		settings := CalendarLabSettings{VisibleCategories: cats, Assignee: "Jordan"}
		got := FilterEvents(events, settings)
		assert.Len(t, got, 2)
		for _, e := range got {
			assert.Equal(t, "Jordan", e.Assignee)
		}
	})

	t.Run("hidden category", func(t *testing.T) {
		catsNoMaint := map[CalendarEventCategory]bool{
			CalCatMaintenance: false,
			CalCatAppointment: true,
			CalCatReminder:    true,
			CalCatDeadline:    true,
		}
		settings := CalendarLabSettings{VisibleCategories: catsNoMaint, Assignee: ""}
		got := FilterEvents(events, settings)
		assert.Len(t, got, 3)
		for _, e := range got {
			assert.NotEqual(t, CalCatMaintenance, e.Category)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		settings := CalendarLabSettings{VisibleCategories: cats, Assignee: ""}
		got := FilterEvents(nil, settings)
		assert.Empty(t, got)
	})
}

func TestCalendarLab_FilteredEventCount(t *testing.T) {
	lab := NewCalendarLab()
	now := time.Now().UTC()

	// Add some events with specific assignees.
	lab.Store.AddEvent(time.Date(now.Year(), now.Month(), 10, 0, 0, 0, 0, time.UTC), "J1", "Jordan", CalCatReminder)
	lab.Store.AddEvent(time.Date(now.Year(), now.Month(), 11, 0, 0, 0, 0, time.UTC), "M1", "Maria", CalCatAppointment)

	totalBefore := lab.EventCount()

	// Filtered count with no filters should equal total.
	assert.Equal(t, totalBefore, lab.FilteredEventCount())

	// Filter to Jordan only.
	lab.UpdateSettings(func(s *CalendarLabSettings) {
		s.Assignee = "Jordan"
	})
	filtered := lab.FilteredEventCount()
	assert.Less(t, filtered, totalBefore,
		"filtering by assignee should reduce count below total")
	assert.GreaterOrEqual(t, filtered, 1, "Jordan should have at least one event")
}

func TestCalendarLab_DayEventsMap(t *testing.T) {
	lab := NewCalendarLab()
	now := time.Now().UTC()

	day10 := time.Date(now.Year(), now.Month(), 10, 0, 0, 0, 0, time.UTC)
	day20 := time.Date(now.Year(), now.Month(), 20, 0, 0, 0, 0, time.UTC)
	lab.Store.AddEvent(day10, "A", "Jordan", CalCatReminder)
	lab.Store.AddEvent(day10, "B", "Maria", CalCatAppointment)
	lab.Store.AddEvent(day20, "C", "Sam", CalCatDeadline)

	m := lab.DayEventsMap()
	assert.GreaterOrEqual(t, len(m[10]), 2, "day 10 should have at least 2 events")
	assert.GreaterOrEqual(t, len(m[20]), 1, "day 20 should have at least 1 event")
}
