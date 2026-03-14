// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/shared"
	"catgoose/dothog/internal/ssebroker"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

type feedRoutes struct {
	actLog *demo.ActivityLog
	broker *ssebroker.SSEBroker
}

func (ar *appRoutes) initFeedRoutes(actLog *demo.ActivityLog, broker *ssebroker.SSEBroker) {
	f := &feedRoutes{actLog: actLog, broker: broker}
	ar.e.GET("/demo/feed", f.handleFeedPage)
	ar.e.GET("/demo/feed/more", f.handleFeedMore)
	ar.e.GET("/sse/activity", f.handleActivitySSE)

	// Seed some initial events so the feed isn't empty on first load.
	seedFeedEvents(actLog)
	// Start background publisher for simulated activity.
	go ar.publishActivityEvents(actLog, broker)
}

func (f *feedRoutes) handleFeedPage(c echo.Context) error {
	events := f.actLog.Recent(20)
	lastID := 0
	if len(events) > 0 {
		lastID = events[len(events)-1].ID
	}
	return handler.RenderBaseLayout(c, views.FeedPage(events, lastID))
}

func (f *feedRoutes) handleFeedMore(c echo.Context) error {
	beforeID, _ := strconv.Atoi(c.QueryParam("before"))
	events := f.actLog.Recent(50)
	// Filter events with ID < beforeID
	var filtered []demo.ActivityEvent
	for _, e := range events {
		if e.ID < beforeID {
			filtered = append(filtered, e)
		}
	}
	// Take last 20
	if len(filtered) > 20 {
		filtered = filtered[:20]
	}
	lastID := 0
	if len(filtered) > 0 {
		lastID = filtered[len(filtered)-1].ID
	}
	hasMore := len(filtered) == 20
	return handler.RenderComponent(c, views.FeedMoreItems(filtered, lastID, hasMore))
}

func (f *feedRoutes) handleActivitySSE(c echo.Context) error {
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	ch, unsub := f.broker.Subscribe(ssebroker.TopicActivityFeed)
	defer unsub()

	ctx := c.Request().Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			fmt.Fprint(c.Response(), msg)
			flusher.Flush()
		}
	}
}

// BroadcastActivity publishes an activity event to the SSE feed.
func BroadcastActivity(broker *ssebroker.SSEBroker, e demo.ActivityEvent) {
	if !broker.HasSubscribers(ssebroker.TopicActivityFeed) {
		return
	}
	buf := statsBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	if err := views.FeedItemOOB(e).Render(shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "broadcast activity"), buf); err != nil {
		statsBufPool.Put(buf)
		return
	}
	msg := ssebroker.NewSSEMessage("activity-event", buf.String()).String()
	statsBufPool.Put(buf)
	broker.Publish(ssebroker.TopicActivityFeed, msg)
}

// --- Simulated activity ---

type activityTemplate struct {
	Action   string
	Resource string
	Names    []string
	Details  []string
}

var activityTemplates = []activityTemplate{
	{"created", "task", []string{"API Endpoint", "Dashboard Widget", "Login Flow", "Report Builder", "Search Index"}, []string{"added to backlog", "assigned to sprint", "created from template"}},
	{"updated", "person", []string{"James Smith", "Mary Johnson", "Robert Williams", "Patricia Brown", "John Jones"}, []string{"profile updated", "department changed", "role changed"}},
	{"moved", "task", []string{"Fix Auth Bug", "Deploy Pipeline", "Code Review", "Database Migration", "UI Redesign"}, []string{"moved to in_progress", "moved to review", "moved to done"}},
	{"approved", "approval", []string{"Travel Request", "Software License", "Training Budget", "New Hire Equipment", "Conference Attendance"}, []string{"$500.00 approved", "$1200.00 approved", "$3000.00 approved"}},
	{"rejected", "approval", []string{"Office Renovation", "Team Offsite", "Hardware Upgrade"}, []string{"$15000.00 rejected", "$8000.00 rejected"}},
	{"updated", "contact", []string{"Alice Reed", "Carol West", "Frank Liu", "Iris Tanaka", "Sam Taylor"}, []string{"contact updated", "email changed", "role changed"}},
	{"deleted", "item", []string{"Old Inventory Item", "Deprecated Widget", "Test Record"}, []string{"removed from catalog", "marked inactive"}},
}

func seedFeedEvents(actLog *demo.ActivityLog) {
	for i := 0; i < 15; i++ {
		tmpl := activityTemplates[rand.IntN(len(activityTemplates))]
		name := tmpl.Names[rand.IntN(len(tmpl.Names))]
		detail := tmpl.Details[rand.IntN(len(tmpl.Details))]
		actLog.Record(tmpl.Action, tmpl.Resource, rand.IntN(50)+1, name, detail)
	}
}

func (ar *appRoutes) publishActivityEvents(actLog *demo.ActivityLog, broker *ssebroker.SSEBroker) {
	for {
		delay := time.Duration(2000+rand.IntN(4000)) * time.Millisecond
		select {
		case <-ar.ctx.Done():
			return
		case <-time.After(delay):
			if !broker.HasSubscribers(ssebroker.TopicActivityFeed) {
				continue
			}
			tmpl := activityTemplates[rand.IntN(len(activityTemplates))]
			name := tmpl.Names[rand.IntN(len(tmpl.Names))]
			detail := tmpl.Details[rand.IntN(len(tmpl.Details))]
			evt := actLog.Record(tmpl.Action, tmpl.Resource, rand.IntN(50)+1, name, detail)
			BroadcastActivity(broker, evt)
		}
	}
}
