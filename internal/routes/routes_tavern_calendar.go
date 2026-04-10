// setup:feature:demo

package routes

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

const (
	topicCalMonth    = "calendar/month"
	topicCalDay      = "calendar/day"
	topicCalStats    = "calendar/stats"
	topicCalActivity = "calendar/activity"
)

type tavernCalendarRoutes struct {
	broker *tavern.SSEBroker
	lab    *demo.CalendarLab
}

func (ar *appRoutes) initTavernCalendarRoutes(broker *tavern.SSEBroker) {
	r := &tavernCalendarRoutes{
		broker: broker,
		lab:    demo.NewCalendarLab(),
	}

	// Replay for activity log so reconnecting clients see recent entries.
	broker.SetReplayPolicy(topicCalActivity, 20)

	ar.e.GET("/realtime/tavern/calendar", r.handlePage)
	ar.e.GET("/sse/tavern/calendar", r.handleSSE)
	ar.e.POST("/realtime/tavern/calendar/day", r.handleSelectDay)
	ar.e.POST("/realtime/tavern/calendar/month", r.handleSetMonth)
	ar.e.POST("/realtime/tavern/calendar/event", r.handleAddEvent)
	ar.e.POST("/realtime/tavern/calendar/event/delete", r.handleDeleteEvent)
	ar.e.POST("/realtime/tavern/calendar/controls", r.handleControls)
	ar.e.POST("/realtime/tavern/calendar/sim/pause", r.handleSimPause)

	broker.RunPublisher(ar.ctx, r.startSimulator)
}

// handlePage renders the full calendar lab page.
func (r *tavernCalendarRoutes) handlePage(c echo.Context) error {
	data := r.buildPageData()
	return handler.RenderBaseLayout(c, views.TavernCalendarPage(data))
}

// handleSSE fans all four calendar topics into one StreamSSE connection.
// Month, day, and stats regions use snapshot-on-subscribe so reconnects
// get the current view immediately. Activity uses replay for recent entries.
func (r *tavernCalendarRoutes) handleSSE(c echo.Context) error {
	monthCh, monthUnsub := r.broker.SubscribeWithSnapshot(topicCalMonth, func() string {
		return r.renderMonthFrame()
	})
	defer monthUnsub()

	dayCh, dayUnsub := r.broker.SubscribeWithSnapshot(topicCalDay, func() string {
		return r.renderDayFrame()
	})
	defer dayUnsub()

	statsCh, statsUnsub := r.broker.SubscribeWithSnapshot(topicCalStats, func() string {
		return r.renderStatsFrame()
	})
	defer statsUnsub()

	lastEventID := c.Request().Header.Get("Last-Event-ID")
	actCh, actUnsub := r.broker.SubscribeFromIDWith(topicCalActivity, lastEventID)
	defer actUnsub()

	ctx := c.Request().Context()
	fanIn := make(chan string, 16)
	go func() {
		defer close(fanIn)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-monthCh:
				if !ok {
					return
				}
				select {
				case fanIn <- msg:
				case <-ctx.Done():
					return
				}
			case msg, ok := <-dayCh:
				if !ok {
					return
				}
				select {
				case fanIn <- msg:
				case <-ctx.Done():
					return
				}
			case msg, ok := <-statsCh:
				if !ok {
					return
				}
				select {
				case fanIn <- msg:
				case <-ctx.Done():
					return
				}
			case msg, ok := <-actCh:
				if !ok {
					return
				}
				select {
				case fanIn <- msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return tavern.StreamSSE(
		ctx,
		c.Response(),
		fanIn,
		func(s string) string { return s },
		tavern.WithStreamHeartbeat(15*time.Second),
	)
}

// handleSelectDay updates the shared selected day and publishes day + month.
func (r *tavernCalendarRoutes) handleSelectDay(c echo.Context) error {
	day := parseDay(c.QueryParam("d"))
	if day.IsZero() {
		return c.String(http.StatusBadRequest, "invalid d")
	}
	r.lab.SelectDay(day)
	r.publishMonth()
	r.publishDay()
	return c.NoContent(http.StatusNoContent)
}

// handleSetMonth navigates to the prev/next month.
func (r *tavernCalendarRoutes) handleSetMonth(c echo.Context) error {
	dir := c.QueryParam("dir")
	year, month := r.lab.Year(), r.lab.Month()
	switch dir {
	case "prev":
		if month == time.January {
			year--
			month = time.December
		} else {
			month--
		}
	case "next":
		if month == time.December {
			year++
			month = time.January
		} else {
			month++
		}
	}
	r.lab.SetMonth(year, month)
	r.publishMonth()
	r.publishDay()
	r.publishStats()
	return c.NoContent(http.StatusNoContent)
}

// handleAddEvent adds a calendar event and publishes all regions.
func (r *tavernCalendarRoutes) handleAddEvent(c echo.Context) error {
	day := parseDay(c.FormValue("d"))
	if day.IsZero() {
		return c.String(http.StatusBadRequest, "invalid d")
	}
	title := c.FormValue("title")
	if title == "" {
		return c.String(http.StatusBadRequest, "title is required")
	}
	cat := demo.CalendarEventCategory(c.FormValue("category"))
	if !validCalendarCategory(cat) {
		cat = demo.CalCatReminder
	}
	r.lab.Store.AddEvent(day, title, "", cat)
	r.lab.RecordActivity(fmt.Sprintf("added \"%s\" on %s", title, day.Format("Jan 2")))
	r.publishAll()
	return c.NoContent(http.StatusNoContent)
}

// handleDeleteEvent removes a calendar event and publishes all regions.
func (r *tavernCalendarRoutes) handleDeleteEvent(c echo.Context) error {
	id, err := strconv.ParseInt(c.QueryParam("id"), 10, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid id")
	}
	r.lab.Store.RemoveEvent(id)
	r.lab.RecordActivity(fmt.Sprintf("deleted event #%d", id))
	r.publishAll()
	return c.NoContent(http.StatusNoContent)
}

// handleControls accepts all sliders/toggles/selects and republishes
// affected regions. The form posts all control values in one shot via
// hx-include="closest .card-body".
func (r *tavernCalendarRoutes) handleControls(c echo.Context) error {
	r.lab.UpdateSettings(func(s *demo.CalendarLabSettings) {
		if v, err := strconv.Atoi(c.FormValue("density")); err == nil && v >= 1 && v <= 8 {
			s.Density = v
		}
		if v, err := strconv.Atoi(c.FormValue("sim_speed")); err == nil && v >= 200 && v <= 5000 {
			s.SimSpeed = v
		}
		if v, err := strconv.Atoi(c.FormValue("burst_size")); err == nil && v >= 1 && v <= 5 {
			s.BurstSize = v
		}
		s.Assignee = c.FormValue("assignee")
		s.CompactMode = c.FormValue("compact") == "on"
		s.HighlightWeekends = c.FormValue("highlight_weekends") == "on"

		// Category toggles: unchecked checkboxes aren't sent, so default to
		// false and only enable the categories present in the form.
		for _, cat := range demo.AllCalendarCategories {
			s.VisibleCategories[cat] = c.FormValue("cat_"+string(cat)) == "on"
		}
	})
	r.publishMonth()
	r.publishDay()
	r.publishStats()
	return c.NoContent(http.StatusNoContent)
}

// handleSimPause toggles the simulator pause state and publishes stats.
func (r *tavernCalendarRoutes) handleSimPause(c echo.Context) error {
	paused := r.lab.TogglePause()
	if paused {
		r.lab.RecordActivity("simulator paused")
	} else {
		r.lab.RecordActivity("simulator resumed")
	}
	r.publishStats()
	r.publishActivity()
	return c.NoContent(http.StatusNoContent)
}

// --- publishers ---

func (r *tavernCalendarRoutes) publishAll() {
	r.publishMonth()
	r.publishDay()
	r.publishStats()
	r.publishActivity()
}

func (r *tavernCalendarRoutes) publishMonth() {
	r.broker.Publish(topicCalMonth, r.renderMonthFrame())
}

func (r *tavernCalendarRoutes) publishDay() {
	r.broker.Publish(topicCalDay, r.renderDayFrame())
}

func (r *tavernCalendarRoutes) publishStats() {
	r.broker.Publish(topicCalStats, r.renderStatsFrame())
}

func (r *tavernCalendarRoutes) publishActivity() {
	html := r.renderActivityFrame()
	r.broker.Publish(topicCalActivity, html)
}

// --- renderers ---

func (r *tavernCalendarRoutes) renderMonthFrame() string {
	data := r.buildMonthData()
	settings := r.lab.Settings()
	return tavern.NewSSEMessage("cal-month",
		renderToString("cal-lab month", views.CalendarLabMonth(data, settings)),
	).String()
}

func (r *tavernCalendarRoutes) renderDayFrame() string {
	selected := r.lab.SelectedDay()
	settings := r.lab.Settings()
	if selected.IsZero() {
		return tavern.NewSSEMessage("cal-day",
			renderToString("cal-lab day empty", views.CalendarLabDayEmpty()),
		).String()
	}
	events := r.lab.Store.EventsForDay(selected)
	return tavern.NewSSEMessage("cal-day",
		renderToString("cal-lab day", views.CalendarLabDay(selected, events, settings)),
	).String()
}

func (r *tavernCalendarRoutes) renderStatsFrame() string {
	settings := r.lab.Settings()
	return tavern.NewSSEMessage("cal-stats",
		renderToString("cal-lab stats", views.CalendarLabStats(r.lab.EventCount(), settings, r.lab.Paused())),
	).String()
}

func (r *tavernCalendarRoutes) renderActivityFrame() string {
	return tavern.NewSSEMessage("cal-activity",
		renderToString("cal-lab activity", views.CalendarLabActivityLog(r.lab.Activity())),
	).String()
}

// --- view-model builders ---

func (r *tavernCalendarRoutes) buildPageData() views.CalendarLabData {
	settings := r.lab.Settings()
	monthData := r.buildMonthData()
	selected := r.lab.SelectedDay()
	var dayEvents []demo.CalendarEvent
	if !selected.IsZero() {
		dayEvents = r.lab.Store.EventsForDay(selected)
	}
	return views.CalendarLabData{
		MonthData:  monthData,
		Settings:   settings,
		DayEvents:  dayEvents,
		Activity:   r.lab.Activity(),
		Paused:     r.lab.Paused(),
		EventCount: r.lab.EventCount(),
	}
}

func (r *tavernCalendarRoutes) buildMonthData() views.CalendarMonthData {
	year, month := r.lab.Year(), r.lab.Month()
	selected := r.lab.SelectedDay()
	events := r.lab.Store.EventsForMonth(year, month)
	counts := r.lab.Store.DayCountsForMonth(year, month)
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

// --- simulator ---

func (r *tavernCalendarRoutes) startSimulator(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if r.lab.Paused() {
				continue
			}
			settings := r.lab.Settings()
			ticker.Reset(time.Duration(settings.SimSpeed) * time.Millisecond)

			actions := r.lab.SimTick()
			for _, a := range actions {
				r.lab.RecordActivity(a)
			}
			r.publishAll()
		}
	}
}
