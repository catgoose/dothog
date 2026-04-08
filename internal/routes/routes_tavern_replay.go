// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
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
	conns  map[string]context.CancelFunc

	// mu guards conns.
	mu sync.Mutex
}

func (ar *appRoutes) initTavernReplayRoutes(broker *tavern.SSEBroker) {
	lab := demo.NewReplayLab(10)
	r := &tavernReplayRoutes{
		broker: broker,
		lab:    lab,
		conns:  make(map[string]context.CancelFunc),
	}

	broker.SetReplayPolicy(TopicTavernReplay, lab.ReplayWindow())

	broker.SetReplayGapPolicy(TopicTavernReplay, tavern.GapFallbackToSnapshot, func() string {
		return renderReplaySnapshot("Replay gap detected: requested events are no longer in the replay window. Showing live events from here.")
	})

	// On reconnect, send debug info to the reconnecting subscriber.
	broker.OnReconnect(TopicTavernReplay, func(info tavern.ReconnectInfo) {
		html := renderReplayDebug(info.LastEventID, info.MissedCount, info.Gap)
		msg := tavern.NewSSEMessage("replay-debug", html).String()
		info.SendToSubscriber(msg)
	})

	ar.e.GET("/realtime/tavern/replay", r.handlePage)
	ar.e.GET("/sse/tavern/replay", r.handleSSE)
	ar.e.POST("/realtime/tavern/replay/emit", r.handleEmit)
	ar.e.POST("/realtime/tavern/replay/burst", r.handleBurst)
	ar.e.POST("/realtime/tavern/replay/window", r.handleWindow)
	ar.e.POST("/realtime/tavern/replay/drop", r.handleDrop)

	broker.RunPublisher(ar.ctx, r.startPublisher)
}

func (r *tavernReplayRoutes) handlePage(c echo.Context) error {
	return handler.RenderBaseLayout(c, views.TavernReplayPage(r.lab.ReplayWindow()))
}

// handleSSE is a custom SSE handler that tracks connections for forced drops.
func (r *tavernReplayRoutes) handleSSE(c echo.Context) error {
	w := c.Response().Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	lastID := c.Request().Header.Get("Last-Event-ID")
	ch, unsub := r.broker.SubscribeFromID(TopicTavernReplay, lastID)
	if ch == nil {
		return c.String(http.StatusServiceUnavailable, "Service Unavailable")
	}
	defer unsub()

	// Create a cancellable context for this connection.
	connID := shared.GenerateContextID()
	ctx, cancel := context.WithCancel(c.Request().Context())
	defer cancel()

	r.mu.Lock()
	r.conns[connID] = cancel
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.conns, connID)
		r.mu.Unlock()
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		return c.String(http.StatusInternalServerError, "Streaming not supported")
	}

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			if _, err := fmt.Fprint(w, msg); err != nil {
				return nil
			}
			flusher.Flush()
		case <-ctx.Done():
			return nil
		}
	}
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

// handleDrop forcibly closes all SSE connections, causing the browser
// to auto-reconnect with Last-Event-ID.
func (r *tavernReplayRoutes) handleDrop(c echo.Context) error {
	r.mu.Lock()
	for id, cancel := range r.conns {
		cancel()
		delete(r.conns, id)
	}
	r.mu.Unlock()
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

func renderReplayDebug(lastEventID string, missedCount int, gap time.Duration) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render replay debug")
	if err := views.ReplayDebug(lastEventID, missedCount, gap).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}
