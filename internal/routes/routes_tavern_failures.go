// setup:feature:demo

package routes

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

const (
	topicFailuresLive = "failures/live"
)

type failuresRoutes struct {
	broker    *tavern.SSEBroker
	counter   atomic.Int64
	latestID  atomic.Value // stores string — most recently published event ID
}

func (ar *appRoutes) initFailuresRoutes(broker *tavern.SSEBroker) {
	r := &failuresRoutes{broker: broker}

	// Tiny replay window so the malformed-Last-Event-ID and expired-window
	// scenarios are easy to trigger.
	broker.SetReplayPolicy(topicFailuresLive, 5)
	broker.SetReplayGapPolicy(topicFailuresLive, tavern.GapFallbackToSnapshot, func() string {
		// Snapshot fallback HTML — published into the result panel when the
		// replay buffer can't satisfy a Last-Event-ID resume.
		return tavern.NewSSEMessage("failures-result",
			renderFailuresReport(views.FailuresReportData{
				Scenario:       "Snapshot fallback",
				Interpretation: "Replay buffer could not satisfy Last-Event-ID.",
				BrokerOutcome:  "Gap policy fired — snapshot fallback applied.",
				Level:          "warning",
			}),
		).String()
	})

	ar.e.GET("/realtime/tavern/failures", r.handlePage)
	ar.e.GET("/sse/tavern/failures", r.handleSSE)
	ar.e.POST("/realtime/tavern/failures/burst", r.handleBurst)
	ar.e.POST("/realtime/tavern/failures/clear-replay", r.handleClearReplay)
	ar.e.POST("/realtime/tavern/failures/scenario/expired", r.handleScenarioExpired)
	ar.e.POST("/realtime/tavern/failures/scenario/gap", r.handleScenarioGap)
	ar.e.GET("/realtime/tavern/failures/latest-id", r.handleLatestID)

	broker.RunPublisher(ar.ctx, r.startBackgroundTrickle)
}

func (r *failuresRoutes) handlePage(c echo.Context) error {
	return handler.RenderBaseLayout(c, views.TavernFailuresPage())
}

// handleSSE serves the failures stream. The Last-Event-ID resume path is
// where most of the failure scenarios live: malformed IDs and IDs outside
// the replay window both go through the gap policy. We read the resume
// hint from either the real Last-Event-ID header or a ?resume= query
// parameter — the latter lets the in-page scenario buttons trigger a
// resume without browser support for setting EventSource headers.
func (r *failuresRoutes) handleSSE(c echo.Context) error {
	lastEventID := c.Request().Header.Get("Last-Event-ID")
	if lastEventID == "" {
		lastEventID = c.QueryParam("resume")
	}

	// SubscribeFromIDWith collapses the fresh-vs-resume branch into a
	// single subscription path. The conditional snapshot frame describing
	// the resume attempt only fires on the resume path, since fresh
	// connections have nothing to explain.
	msgs, unsub := r.broker.SubscribeFromIDWith(topicFailuresLive, lastEventID)
	defer unsub()

	var opts []tavern.StreamSSEOption
	if lastEventID != "" {
		opts = append(opts, tavern.WithStreamSnapshot(func() string {
			return resumeDescriptionFrame(lastEventID)
		}))
	}

	return tavern.StreamSSE(
		c.Request().Context(),
		c.Response(),
		msgs,
		func(s string) string { return s },
		opts...,
	)
}

// handleBurst publishes a few events with sequential IDs so the replay
// window has something to (over)flow.
func (r *failuresRoutes) handleBurst(c echo.Context) error {
	r.publishBurst(8)
	return c.NoContent(http.StatusNoContent)
}

func (r *failuresRoutes) publishBurst(n int) {
	for i := 0; i < n; i++ {
		r.publishOne()
	}
}

func (r *failuresRoutes) handleClearReplay(c echo.Context) error {
	r.broker.ClearReplay(topicFailuresLive)
	r.broker.Publish(topicFailuresLive,
		tavern.NewSSEMessage("failures-result",
			renderFailuresReport(views.FailuresReportData{
				Scenario:       "Clear replay buffer",
				Interpretation: "No resume attempted.",
				BrokerOutcome:  "Replay buffer emptied. Next Last-Event-ID resume will gap-fallback.",
				Level:          "info",
			}),
		).String(),
	)
	return c.NoContent(http.StatusNoContent)
}

// handleScenarioExpired deterministically demonstrates an expired replay
// window. It captures the current latest ID, publishes enough events to
// overflow the 5-event replay window, then returns the captured ID so
// the client can reconnect with it.
func (r *failuresRoutes) handleScenarioExpired(c echo.Context) error {
	capturedID := r.getLatestID()
	if capturedID == "" {
		// No events yet — publish one so we have something to expire.
		r.publishOne()
		capturedID = r.getLatestID()
	}
	// Overflow the replay window (size=5) so the captured ID is gone.
	r.publishBurst(8)
	return c.JSON(http.StatusOK, map[string]string{"resume_id": capturedID})
}

// handleScenarioGap deterministically triggers a replay gap fallback.
// It captures the current latest ID, publishes enough to overflow the
// replay window, clears the replay buffer entirely, then returns the
// captured ID for the client to reconnect with.
func (r *failuresRoutes) handleScenarioGap(c echo.Context) error {
	capturedID := r.getLatestID()
	if capturedID == "" {
		r.publishOne()
		capturedID = r.getLatestID()
	}
	r.publishBurst(8)
	r.broker.ClearReplay(topicFailuresLive)
	return c.JSON(http.StatusOK, map[string]string{"resume_id": capturedID})
}

// handleLatestID returns the most recently published event ID so the
// client can display it and use it for deterministic scenarios.
func (r *failuresRoutes) handleLatestID(c echo.Context) error {
	id := r.getLatestID()
	return c.JSON(http.StatusOK, map[string]string{"latest_id": id})
}

func (r *failuresRoutes) getLatestID() string {
	v := r.latestID.Load()
	if v == nil {
		return ""
	}
	return v.(string)
}

// publishOne publishes a single event and stores the ID as latest.
func (r *failuresRoutes) publishOne() {
	seq := r.counter.Add(1)
	id := fmt.Sprintf("evt-%d", seq)
	html := renderFailuresEvent(seq, id, time.Now().Format("15:04:05.000"))
	msg := tavern.NewSSEMessage("failures-event", html).WithID(id).String()
	r.broker.PublishWithID(topicFailuresLive, id, msg)
	r.latestID.Store(id)
}

// startBackgroundTrickle keeps a slow trickle of events going so the
// page is never empty.
func (r *failuresRoutes) startBackgroundTrickle(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.publishOne()
		}
	}
}

// resumeDescriptionFrame returns a one-shot SSE result frame describing how
// the resume attempt was interpreted. The client renders this in the result
// panel. It is delivered via tavern.WithStreamSnapshot before any live events.
func resumeDescriptionFrame(lastEventID string) string {
	var d views.FailuresReportData
	d.ResumeToken = lastEventID
	if !looksLikeFailuresEventID(lastEventID) {
		d.Scenario = "Malformed Last-Event-ID"
		d.Interpretation = fmt.Sprintf("Token %q does not match the evt-N scheme.", lastEventID)
		d.BrokerOutcome = "Replay lookup still attempted. Missing IDs trigger the gap policy."
		d.Level = "warning"
	} else {
		d.Scenario = "Resume attempted"
		d.Interpretation = fmt.Sprintf("Token %q is valid-shaped and was previously issued by this lab.", lastEventID)
		d.BrokerOutcome = "Looked up in replay buffer. If expired, the gap policy fires (snapshot fallback)."
		d.Level = "info"
	}
	html := renderFailuresReport(d)
	return tavern.NewSSEMessage("failures-result", html).String()
}

// looksLikeFailuresEventID is a tiny shape check matching IDs this lab emits.
func looksLikeFailuresEventID(id string) bool {
	if len(id) < 5 || id[:4] != "evt-" {
		return false
	}
	for _, c := range id[4:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func renderFailuresEvent(seq int64, id, timestamp string) string {
	return renderToString("render failures event", views.FailuresEvent(seq, id, timestamp))
}

func renderFailuresReport(d views.FailuresReportData) string {
	return renderToString("render failures report", views.FailuresReport(d))
}
