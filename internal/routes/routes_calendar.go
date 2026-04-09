// setup:feature:demo

package routes

import (
	"net/http"
	"strconv"
	"time"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

type calendarRoutes struct {
	store *demo.CalendarStore
}

func (ar *appRoutes) initCalendarRoutes() {
	c := &calendarRoutes{store: demo.NewCalendarStore()}

	ar.e.GET("/realtime/calendar", c.handlePage)
	ar.e.GET("/realtime/calendar/month", c.handleMonth)
	ar.e.GET("/realtime/calendar/day", c.handleDay)
	ar.e.POST("/realtime/calendar/event", c.handleAddEvent)
	ar.e.DELETE("/realtime/calendar/event/:id", c.handleDeleteEvent)
}

func (c *calendarRoutes) handlePage(ctx echo.Context) error {
	year, month := parseYearMonth(ctx.QueryParam("y"), ctx.QueryParam("m"))
	day := parseDay(ctx.QueryParam("d"))
	data := c.buildMonthData(year, month, day)
	return handler.RenderBaseLayout(ctx, views.CalendarPage(data))
}

// handleMonth returns just the month grid fragment for prev/next navigation.
func (c *calendarRoutes) handleMonth(ctx echo.Context) error {
	year, month := parseYearMonth(ctx.QueryParam("y"), ctx.QueryParam("m"))
	day := parseDay(ctx.QueryParam("d"))
	data := c.buildMonthData(year, month, day)
	return handler.RenderComponent(ctx, views.CalendarMonthFragment(data))
}

// handleDay returns the day inspector panel for the selected day.
func (c *calendarRoutes) handleDay(ctx echo.Context) error {
	day := parseDay(ctx.QueryParam("d"))
	if day.IsZero() {
		return ctx.String(http.StatusBadRequest, "missing or invalid d")
	}
	events := c.store.EventsForDay(day)
	return handler.RenderComponent(ctx, views.CalendarDayPanel(day, events))
}

func (c *calendarRoutes) handleAddEvent(ctx echo.Context) error {
	day := parseDay(ctx.FormValue("d"))
	if day.IsZero() {
		return ctx.String(http.StatusBadRequest, "missing or invalid d")
	}
	title := ctx.FormValue("title")
	if title == "" {
		return ctx.String(http.StatusBadRequest, "title is required")
	}
	cat := demo.CalendarEventCategory(ctx.FormValue("category"))
	if !validCalendarCategory(cat) {
		cat = demo.CalCatReminder
	}
	assignee := ctx.FormValue("assignee")
	c.store.AddEvent(day, title, assignee, cat)

	// Re-render both the day panel (where the form lives) and the month grid
	// (so the new dot indicator shows up). Use OOB swap for the month grid.
	events := c.store.EventsForDay(day)
	monthData := c.buildMonthData(day.Year(), day.Month(), day)
	return handler.RenderComponent(ctx, views.CalendarDayWithMonthOOB(day, events, monthData))
}

func (c *calendarRoutes) handleDeleteEvent(ctx echo.Context) error {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		return ctx.String(http.StatusBadRequest, "invalid id")
	}
	c.store.RemoveEvent(id)

	// Day on which the deletion happened comes from the form/query so we can
	// re-render the day panel and month grid.
	day := parseDay(ctx.QueryParam("d"))
	if day.IsZero() {
		return ctx.NoContent(http.StatusNoContent)
	}
	events := c.store.EventsForDay(day)
	monthData := c.buildMonthData(day.Year(), day.Month(), day)
	return handler.RenderComponent(ctx, views.CalendarDayWithMonthOOB(day, events, monthData))
}

func (c *calendarRoutes) buildMonthData(year int, month time.Month, selected time.Time) views.CalendarMonthData {
	events := c.store.EventsForMonth(year, month)
	counts := c.store.DayCountsForMonth(year, month)
	return views.CalendarMonthData{
		Year:        year,
		Month:       month,
		Today:       time.Now().UTC(),
		Selected:    selected,
		Events:      events,
		DayCounts:   counts,
		PrevYear:    prevMonthYear(year, month),
		PrevMonth:   prevMonth(month),
		NextYear:    nextMonthYear(year, month),
		NextMonth:   nextMonth(month),
		WeekStart:   firstWeekStart(year, month),
		WeeksInView: weeksInMonthView(year, month),
	}
}

// parseYearMonth parses query params, defaulting to the current month.
func parseYearMonth(yStr, mStr string) (int, time.Month) {
	now := time.Now().UTC()
	year := now.Year()
	month := now.Month()
	if y, err := strconv.Atoi(yStr); err == nil && y > 1900 && y < 3000 {
		year = y
	}
	if m, err := strconv.Atoi(mStr); err == nil && m >= 1 && m <= 12 {
		month = time.Month(m)
	}
	return year, month
}

// parseDay parses a YYYY-MM-DD string. Returns zero time if invalid.
func parseDay(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func validCalendarCategory(c demo.CalendarEventCategory) bool {
	for _, x := range demo.AllCalendarCategories {
		if x == c {
			return true
		}
	}
	return false
}

func prevMonth(m time.Month) time.Month {
	if m == time.January {
		return time.December
	}
	return m - 1
}

func nextMonth(m time.Month) time.Month {
	if m == time.December {
		return time.January
	}
	return m + 1
}

func prevMonthYear(y int, m time.Month) int {
	if m == time.January {
		return y - 1
	}
	return y
}

func nextMonthYear(y int, m time.Month) int {
	if m == time.December {
		return y + 1
	}
	return y
}

// firstWeekStart returns the Sunday on or before the first of the month.
func firstWeekStart(year int, month time.Month) time.Time {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	offset := int(first.Weekday()) // Sunday=0
	return first.AddDate(0, 0, -offset)
}

// weeksInMonthView returns how many week-rows the month grid needs (5 or 6).
func weeksInMonthView(year int, month time.Month) int {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1)
	startOffset := int(first.Weekday())
	totalCells := startOffset + last.Day()
	weeks := totalCells / 7
	if totalCells%7 != 0 {
		weeks++
	}
	return weeks
}
