// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/shared"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

type tavernReplayRoutes struct {
	broker         *tavern.SSEBroker
	lab            *demo.ReplayLab
	lifetime       atomic.Int64 // nanoseconds; 0 = no limit
	reconnectDelay atomic.Int64 // nanoseconds; 0 = default (1s)
	publishRate    atomic.Int64 // nanoseconds
}

func (ar *appRoutes) initTavernReplayRoutes(broker *tavern.SSEBroker) {
	lab := demo.NewReplayLab(50)
	r := &tavernReplayRoutes{broker: broker, lab: lab}
	r.lifetime.Store(int64(10 * time.Second))
	r.reconnectDelay.Store(int64(5 * time.Second))
	r.publishRate.Store(int64(2 * time.Second))

	broker.SetReplayPolicy(TopicTavernReplay, lab.ReplayWindow())

	broker.SetReplayGapPolicy(TopicTavernReplay, tavern.GapFallbackToSnapshot, func() string {
		return renderReplaySnapshot("Replay gap detected: requested events are no longer in the replay window. Showing live events from here.")
	})

	// On reconnect, publish debug info so the UI shows replay stats.
	// Uses Publish (broadcast) instead of SendToSubscriber because the
	// callback fires in a goroutine and SendToSubscriber is non-blocking —
	// the message is silently dropped if the channel buffer is full.
	broker.OnReconnect(TopicTavernReplay, func(info tavern.ReconnectInfo) {
		gapDetected := info.LastEventID != "" && info.Gap == 0
		html := renderReplayDebug(info.LastEventID, info.ReplayDelivered, info.ReplayDropped, info.Gap, gapDetected)
		msg := tavern.NewSSEMessage("replay-debug", html).String()
		broker.Publish(TopicTavernReplay, msg)
	})

	ar.e.GET("/realtime/tavern/replay", r.handlePage)
	ar.e.GET("/sse/tavern/replay", r.handleSSE)
	ar.e.POST("/realtime/tavern/replay/emit", r.handleEmit)
	ar.e.POST("/realtime/tavern/replay/burst", r.handleBurst)
	ar.e.POST("/realtime/tavern/replay/window", r.handleWindow)
	ar.e.POST("/realtime/tavern/replay/lifetime", r.handleLifetime)
	ar.e.POST("/realtime/tavern/replay/delay", r.handleDelay)
	ar.e.POST("/realtime/tavern/replay/rate", r.handleRate)
	ar.e.POST("/realtime/tavern/replay/preset", r.handlePreset)

	broker.RunPublisher(ar.ctx, r.startPublisher)
}

func (r *tavernReplayRoutes) handlePage(c echo.Context) error {
	lt := time.Duration(r.lifetime.Load())
	rd := time.Duration(r.reconnectDelay.Load())
	pr := time.Duration(r.publishRate.Load())
	return handler.RenderBaseLayout(c, views.TavernReplayPage(r.lab.ReplayWindow(), lt, rd, pr))
}

// handleSSE delegates to tavern's built-in SSEHandler with the current
// max connection duration and reconnect delay. Each new connection picks
// up the latest settings.
func (r *tavernReplayRoutes) handleSSE(c echo.Context) error {
	lt := time.Duration(r.lifetime.Load())
	rd := time.Duration(r.reconnectDelay.Load())
	var opts []tavern.SSEHandlerOption
	if lt > 0 {
		opts = append(opts, tavern.WithMaxConnectionDuration(lt))
	}
	if rd > 0 {
		opts = append(opts, tavern.WithReconnectDelay(rd))
	}
	h := r.broker.SSEHandler(TopicTavernReplay, opts...)
	h.ServeHTTP(c.Response().Writer, c.Request())
	return nil
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

func (r *tavernReplayRoutes) handleLifetime(c echo.Context) error {
	s, err := strconv.Atoi(c.FormValue("seconds"))
	if err != nil || s < 1 {
		return c.String(http.StatusBadRequest, "invalid lifetime")
	}
	r.lifetime.Store(int64(time.Duration(s) * time.Second))
	return c.HTML(http.StatusOK, fmt.Sprintf("%ds", s))
}

func (r *tavernReplayRoutes) handleDelay(c echo.Context) error {
	s, err := strconv.Atoi(c.FormValue("seconds"))
	if err != nil || s < 0 {
		return c.String(http.StatusBadRequest, "invalid delay")
	}
	r.reconnectDelay.Store(int64(time.Duration(s) * time.Second))
	return c.HTML(http.StatusOK, fmt.Sprintf("%ds", s))
}

func (r *tavernReplayRoutes) handleRate(c echo.Context) error {
	ms, err := strconv.Atoi(c.FormValue("ms"))
	if err != nil || ms < 100 {
		return c.String(http.StatusBadRequest, "invalid rate")
	}
	r.publishRate.Store(int64(time.Duration(ms) * time.Millisecond))
	return c.HTML(http.StatusOK, formatRateLabel(ms))
}

func (r *tavernReplayRoutes) handlePreset(c echo.Context) error {
	var window int
	switch c.FormValue("preset") {
	case "replay":
		window = 50
	case "gap":
		window = 5
	default:
		return c.String(http.StatusBadRequest, "unknown preset")
	}
	r.lab.SetReplayWindow(window)
	r.broker.SetReplayPolicy(TopicTavernReplay, window)
	r.lifetime.Store(int64(3 * time.Second))
	r.reconnectDelay.Store(int64(5 * time.Second))
	r.publishRate.Store(int64(2 * time.Second))
	// Tell the client to sync slider values.
	c.Response().Header().Set("HX-Trigger", fmt.Sprintf(
		`{"replay-preset":{"window":%d,"lifetime":3,"delay":5,"rate":2000}}`, window))
	return c.NoContent(http.StatusNoContent)
}

func formatRateLabel(ms int) string {
	if ms >= 1000 {
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	}
	return fmt.Sprintf("%dms", ms)
}

func (r *tavernReplayRoutes) publishEvent() {
	id, seq := r.lab.NextEvent()
	ts := time.Now().Format("15:04:05")
	html := renderReplayEvent(seq, id, ts)
	msg := tavern.NewSSEMessage("replay-event", html).String()
	r.broker.PublishWithID(TopicTavernReplay, id, msg)
}

func (r *tavernReplayRoutes) startPublisher(ctx context.Context) {
	rate := time.Duration(r.publishRate.Load())
	ticker := time.NewTicker(rate)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.publishEvent()
			if cur := time.Duration(r.publishRate.Load()); cur != rate {
				rate = cur
				ticker.Reset(rate)
			}
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

func renderReplayDebug(lastEventID string, delivered, dropped int, gap time.Duration, gapDetected bool) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render replay debug")
	if err := views.ReplayDebug(lastEventID, delivered, dropped, gap, gapDetected).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}
