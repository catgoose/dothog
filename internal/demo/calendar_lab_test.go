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
