// setup:feature:demo

package routes

import (
	"context"
	"encoding/json"
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
	topicToastEvents = "toastlab/events"
	topicToastStats  = "toastlab/stats"
	topicToastLog    = "toastlab/log"
)

type tavernToastRoutes struct {
	broker *tavern.SSEBroker
	lab    *demo.ToastLab
}

func (ar *appRoutes) initTavernToastRoutes(broker *tavern.SSEBroker) {
	r := &tavernToastRoutes{
		broker: broker,
		lab:    demo.NewToastLab(),
	}

	broker.SetReplayPolicy(topicToastLog, 20)

	ar.e.GET("/realtime/tavern/toasts", r.handlePage)
	ar.e.GET("/sse/tavern/toasts", r.handleSSE)
	ar.e.POST("/realtime/tavern/toasts/controls", r.handleControls)
	ar.e.POST("/realtime/tavern/toasts/pause", r.handlePause)
	ar.e.POST("/realtime/tavern/toasts/reset", r.handleReset)
	ar.e.POST("/realtime/tavern/toasts/emit", r.handleEmit)
	ar.e.POST("/realtime/tavern/toasts/burst", r.handleBurst)

	broker.RunPublisher(ar.ctx, r.startSimulator)
}

func (r *tavernToastRoutes) handlePage(c echo.Context) error {
	data := views.ToastLabData{
		Settings: r.lab.Settings(),
		Stats:    r.lab.Stats(),
		Log:      r.lab.LifecycleLog(),
		Paused:   r.lab.Paused(),
	}
	return handler.RenderBaseLayout(c, views.ToastLabPage(data))
}

func (r *tavernToastRoutes) handleSSE(c echo.Context) error {
	eventsCh, eventsUnsub := r.broker.Subscribe(topicToastEvents)
	defer eventsUnsub()

	statsCh, statsUnsub := r.broker.SubscribeWithSnapshot(topicToastStats, func() string {
		return r.renderStatsFrame()
	})
	defer statsUnsub()

	lastEventID := c.Request().Header.Get("Last-Event-ID")
	logCh, logUnsub := r.broker.SubscribeFromIDWith(topicToastLog, lastEventID)
	defer logUnsub()

	ctx := c.Request().Context()
	fanIn := make(chan string, 20)
	go func() {
		defer close(fanIn)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			for {
				select {
				case msg, ok := <-eventsCh:
					if !ok {
						return
					}
					select {
					case fanIn <- msg:
					case <-ctx.Done():
						return
					}
				default:
					goto drainStats
				}
			}
		drainStats:
			for {
				select {
				case msg, ok := <-statsCh:
					if !ok {
						return
					}
					select {
					case fanIn <- msg:
					case <-ctx.Done():
						return
					}
				default:
					goto drainLog
				}
			}
		drainLog:
			for {
				select {
				case msg, ok := <-logCh:
					if !ok {
						return
					}
					select {
					case fanIn <- msg:
					case <-ctx.Done():
						return
					}
				default:
					goto done
				}
			}
		done:
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

func (r *tavernToastRoutes) handleControls(c echo.Context) error {
	r.lab.UpdateSettings(func(s *demo.ToastLabSettings) {
		if v, err := strconv.Atoi(c.FormValue("rate_ms")); err == nil && v >= 100 && v <= 5000 {
			s.RateMS = v
		}
		if v, err := strconv.Atoi(c.FormValue("dismiss_dur")); err == nil && v >= 1000 && v <= 30000 {
			s.DismissDurMS = v
		}
		if v, err := strconv.Atoi(c.FormValue("stack_size")); err == nil && v >= 3 && v <= 20 {
			s.StackSize = v
		}
		if v, err := strconv.Atoi(c.FormValue("burst_size")); err == nil && v >= 1 && v <= 10 {
			s.BurstSize = v
		}
		mix := demo.ToastSeverityMix(c.FormValue("severity_mix"))
		if mix == demo.ToastMixBalanced || mix == demo.ToastMixWarningHeavy || mix == demo.ToastMixErrorHeavy || mix == demo.ToastMixSuccessOnly {
			s.SeverityMix = mix
		}
		s.ReplayRecent = c.FormValue("replay_recent") == "on"
	})
	r.publishStats()
	return c.NoContent(http.StatusNoContent)
}

func (r *tavernToastRoutes) handlePause(c echo.Context) error {
	paused := r.lab.TogglePause()
	if paused {
		r.lab.RecordLifecycle("simulator paused")
	} else {
		r.lab.RecordLifecycle("simulator resumed")
	}
	r.publishStats()
	r.publishLog()
	return c.NoContent(http.StatusNoContent)
}

func (r *tavernToastRoutes) handleReset(c echo.Context) error {
	r.lab.ResetStats()
	r.lab.RecordLifecycle("stats reset")
	r.publishStats()
	r.publishLog()
	return c.NoContent(http.StatusNoContent)
}

func (r *tavernToastRoutes) handleEmit(c echo.Context) error {
	sev := demo.ToastSeverity(c.QueryParam("severity"))
	if sev == "" {
		sev = demo.ToastInfo
	}
	evt := r.lab.Emit(sev)
	r.lab.RecordLifecycle("manual: " + evt.Title)
	r.publishEvent(evt)
	r.publishStats()
	r.publishLog()
	return c.NoContent(http.StatusNoContent)
}

func (r *tavernToastRoutes) handleBurst(c echo.Context) error {
	events := r.lab.SimTick()
	for _, evt := range events {
		r.lab.RecordLifecycle("burst: " + evt.Title)
		r.publishEvent(evt)
	}
	r.publishStats()
	r.publishLog()
	return c.NoContent(http.StatusNoContent)
}

// --- publishers ---

func (r *tavernToastRoutes) publishEvent(evt demo.ToastEvent) {
	data, _ := json.Marshal(evt)
	msg := tavern.NewSSEMessage("toast-event", string(data)).String()
	r.broker.Publish(topicToastEvents, msg)
}

func (r *tavernToastRoutes) publishStats() {
	r.broker.Publish(topicToastStats, r.renderStatsFrame())
}

func (r *tavernToastRoutes) publishLog() {
	html := renderToString("toast-lab log", views.ToastLabLogEntries(r.lab.LifecycleLog()))
	r.broker.Publish(topicToastLog, tavern.NewSSEMessage("toast-log", html).String())
}

func (r *tavernToastRoutes) renderStatsFrame() string {
	settings := r.lab.Settings()
	stats := r.lab.Stats()
	return tavern.NewSSEMessage("toast-stats",
		renderToString("toast-lab stats", views.ToastLabStatsFragment(settings, stats, r.lab.Paused())),
	).String()
}

// --- simulator ---

func (r *tavernToastRoutes) startSimulator(ctx context.Context) {
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
			ticker.Reset(time.Duration(settings.RateMS) * time.Millisecond)

			events := r.lab.SimTick()
			for _, evt := range events {
				r.lab.RecordLifecycle("sim: " + evt.Title)
				r.publishEvent(evt)
			}
			r.publishStats()
			r.publishLog()
		}
	}
}
