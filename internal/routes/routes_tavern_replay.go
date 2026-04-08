// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/shared"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

type tavernReplayRoutes struct {
	broker *tavern.SSEBroker
	lab    *demo.ReplayLab
}

func (ar *appRoutes) initTavernReplayRoutes(broker *tavern.SSEBroker) {
	lab := demo.NewReplayLab(10)
	r := &tavernReplayRoutes{broker: broker, lab: lab}

	broker.SetReplayPolicy(TopicTavernReplay, lab.ReplayWindow())

	broker.SetReplayGapPolicy(TopicTavernReplay, tavern.GapFallbackToSnapshot, func() string {
		return renderReplaySnapshot("Replay gap detected: requested events are no longer in the replay window. Showing live events from here.")
	})

	ar.e.GET("/realtime/tavern/replay", r.handlePage)
	ar.e.GET("/sse/tavern/replay", echo.WrapHandler(broker.SSEHandler(TopicTavernReplay)))
	ar.e.POST("/realtime/tavern/replay/emit", r.handleEmit)
	ar.e.POST("/realtime/tavern/replay/burst", r.handleBurst)
	ar.e.POST("/realtime/tavern/replay/window", r.handleWindow)
	ar.e.POST("/realtime/tavern/replay/disconnect", r.handleDisconnect)

	broker.RunPublisher(ar.ctx, r.startPublisher)
}

func (r *tavernReplayRoutes) handlePage(c echo.Context) error {
	return handler.RenderBaseLayout(c, views.TavernReplayPage(r.lab.ReplayWindow()))
}

func (r *tavernReplayRoutes) handleEmit(c echo.Context) error {
	r.publishEvent()
	return c.NoContent(http.StatusNoContent)
}

func (r *tavernReplayRoutes) handleBurst(c echo.Context) error {
	for range 30 {
		r.publishEvent()
	}
	return c.NoContent(http.StatusNoContent)
}

func (r *tavernReplayRoutes) handleWindow(c echo.Context) error {
	n, err := strconv.Atoi(c.FormValue("window"))
	if err != nil || n < 1 {
		return c.String(http.StatusBadRequest, "invalid window")
	}
	r.lab.SetReplayWindow(n)
	r.broker.SetReplayPolicy(TopicTavernReplay, n)
	return c.HTML(http.StatusOK, fmt.Sprintf("%d", n))
}

func (r *tavernReplayRoutes) handleDisconnect(c echo.Context) error {
	// Publish a close event — the sse-close="replay-close" attribute on the
	// client container tells the HTMX SSE extension to close the EventSource
	// when it receives an event with this name.
	msg := tavern.NewSSEMessage("replay-close", "").String()
	r.broker.Publish(TopicTavernReplay, msg)
	return c.NoContent(http.StatusNoContent)
}

func (r *tavernReplayRoutes) publishEvent() {
	id, seq := r.lab.NextEvent()
	ts := time.Now().Format("15:04:05")
	html := renderReplayEvent(seq, id, ts)
	msg := tavern.NewSSEMessage("replay-event", html).WithID(id).String()
	r.broker.Publish(TopicTavernReplay, msg)
}

func (r *tavernReplayRoutes) startPublisher(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !r.broker.HasSubscribers(TopicTavernReplay) {
				continue
			}
			r.publishEvent()
		}
	}
}

func renderReplayEvent(seq int64, id, timestamp string) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render replay event")
	if err := views.ReplayEvent(seq, id, timestamp).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}

func renderReplaySnapshot(message string) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render replay snapshot")
	if err := views.ReplaySnapshot(message).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}
