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
}

func (ar *appRoutes) initTavernReplayRoutes(broker *tavern.SSEBroker) {
	lab := demo.NewReplayLab(10)
	r := &tavernReplayRoutes{broker: broker, lab: lab}
	r.lifetime.Store(int64(30 * time.Second))
	r.reconnectDelay.Store(int64(1 * time.Second))

	broker.SetReplayPolicy(TopicTavernReplay, lab.ReplayWindow())

	broker.SetReplayGapPolicy(TopicTavernReplay, tavern.GapFallbackToSnapshot, func() string {
		return renderReplaySnapshot("Replay gap detected: requested events are no longer in the replay window. Showing live events from here.")
	})

	// On reconnect, send debug info to the reconnecting subscriber.
	// When the Last-Event-ID is not found in the replay log (gap),
	// tavern sets Gap=0 and MissedCount=0. A non-empty LastEventID
	// with Gap=0 reliably indicates the ID has rolled out.
	broker.OnReconnect(TopicTavernReplay, func(info tavern.ReconnectInfo) {
		gapDetected := info.LastEventID != "" && info.Gap == 0
		html := renderReplayDebug(info.LastEventID, info.MissedCount, info.Gap, gapDetected)
		msg := tavern.NewSSEMessage("replay-debug", html).String()
		info.SendToSubscriber(msg)
	})

	ar.e.GET("/realtime/tavern/replay", r.handlePage)
	ar.e.GET("/sse/tavern/replay", r.handleSSE)
	ar.e.POST("/realtime/tavern/replay/emit", r.handleEmit)
	ar.e.POST("/realtime/tavern/replay/burst", r.handleBurst)
	ar.e.POST("/realtime/tavern/replay/window", r.handleWindow)
	ar.e.POST("/realtime/tavern/replay/lifetime", r.handleLifetime)
	ar.e.POST("/realtime/tavern/replay/delay", r.handleDelay)

	broker.RunPublisher(ar.ctx, r.startPublisher)
}

func (r *tavernReplayRoutes) handlePage(c echo.Context) error {
	lt := time.Duration(r.lifetime.Load())
	rd := time.Duration(r.reconnectDelay.Load())
	return handler.RenderBaseLayout(c, views.TavernReplayPage(r.lab.ReplayWindow(), lt, rd))
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

func renderReplayDebug(lastEventID string, missedCount int, gap time.Duration, gapDetected bool) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render replay debug")
	if err := views.ReplayDebug(lastEventID, missedCount, gap, gapDetected).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}
