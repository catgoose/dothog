// setup:feature:demo

package routes

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

const (
	topicHZStats    = "hz/stats"
	topicHZActivity = "hz/activity"
)

func topicHZRegion(id int) string {
	return fmt.Sprintf("hz/region/%d", id)
}

type tavernHotZoneRoutes struct {
	broker *tavern.SSEBroker
	lab    *demo.HotZoneLab
}

func (ar *appRoutes) initTavernHotZoneRoutes(broker *tavern.SSEBroker) {
	r := &tavernHotZoneRoutes{
		broker: broker,
		lab:    demo.NewHotZoneLab(),
	}

	broker.SetReplayPolicy(topicHZActivity, 10)

	ar.e.GET("/realtime/tavern/hotzones", r.handlePage)
	ar.e.GET("/sse/tavern/hotzones", r.handleSSE)
	ar.e.POST("/realtime/tavern/hotzones/controls", r.handleControls)
	ar.e.POST("/realtime/tavern/hotzones/pause", r.handlePause)
	ar.e.POST("/realtime/tavern/hotzones/reset", r.handleReset)
	ar.e.POST("/realtime/tavern/hotzones/command", r.handleCommand)
	ar.e.POST("/realtime/tavern/hotzones/lifecycle", r.handleLifecycle)

	broker.RunPublisher(ar.ctx, r.startSimulator)
}

func (r *tavernHotZoneRoutes) handlePage(c echo.Context) error {
	return handler.RenderBaseLayout(c, views.HotZoneLabPage(r.buildPageData()))
}

func (r *tavernHotZoneRoutes) handleSSE(c echo.Context) error {
	settings := r.lab.Settings()

	type sub struct {
		ch    <-chan string
		unsub func()
	}
	regionSubs := make([]sub, settings.RegionCount)
	for i := 0; i < settings.RegionCount; i++ {
		id := i + 1
		ch, unsub := r.broker.SubscribeWithSnapshot(topicHZRegion(id), func() string {
			return r.renderRegionFrame(id)
		})
		regionSubs[i] = sub{ch, unsub}
	}
	defer func() {
		for _, s := range regionSubs {
			s.unsub()
		}
	}()

	statsCh, statsUnsub := r.broker.SubscribeWithSnapshot(topicHZStats, func() string {
		return r.renderStatsFrame()
	})
	defer statsUnsub()

	lastEventID := c.Request().Header.Get("Last-Event-ID")
	actCh, actUnsub := r.broker.SubscribeFromIDWith(topicHZActivity, lastEventID)
	defer actUnsub()

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
			for _, s := range regionSubs {
				for {
					select {
					case msg, ok := <-s.ch:
						if !ok {
							return
						}
						select {
						case fanIn <- msg:
						case <-ctx.Done():
							return
						}
					default:
						goto nextRegion
					}
				}
			nextRegion:
			}
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
					goto drainAct
				}
			}
		drainAct:
			for {
				select {
				case msg, ok := <-actCh:
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

func (r *tavernHotZoneRoutes) handleControls(c echo.Context) error {
	r.lab.UpdateSettings(func(s *demo.HotZoneSettings) {
		if preset := demo.HotZonePreset(c.FormValue("preset")); preset != "" {
			switch preset {
			case demo.HotZonePresetNormal, demo.HotZonePresetHot, demo.HotZonePresetNasty, demo.HotZonePresetHell:
				s.ApplyPreset(preset)
				return
			}
		}
		s.Preset = ""
		if v, err := strconv.Atoi(c.FormValue("update_interval")); err == nil && v >= 25 && v <= 5000 {
			s.UpdateIntervalMS = v
		}
		if v, err := strconv.Atoi(c.FormValue("region_count")); err == nil && v >= 1 && v <= 64 {
			s.RegionCount = v
		}
		if v, err := strconv.Atoi(c.FormValue("grid_size")); err == nil && v >= 2 && v <= 16 {
			s.GridSize = v
		}
		if v, err := strconv.Atoi(c.FormValue("focused_region")); err == nil && v >= 0 && v <= 64 {
			s.FocusedRegion = v
		}
		if v, err := strconv.Atoi(c.FormValue("jitter_min")); err == nil && v >= 0 && v <= 2000 {
			s.JitterMinMS = v
		}
		if v, err := strconv.Atoi(c.FormValue("jitter_max")); err == nil && v >= 0 && v <= 5000 {
			s.JitterMaxMS = v
		}
		s.BurstMode = c.FormValue("burst_mode") == "on"
		mode := demo.HotZoneMode(c.FormValue("command_mode"))
		if mode == demo.HotZoneModeHXPost || mode == demo.HotZoneModeTavern {
			s.CommandMode = mode
		}
		scope := demo.HotZoneSwapScope(c.FormValue("swap_scope"))
		if scope == demo.HotZoneSwapInner || scope == demo.HotZoneSwapCard {
			s.SwapScope = scope
		}
		// Heat-map controls.
		s.HeatEnabled = c.FormValue("heat_enabled") == "on"
		if v, err := strconv.Atoi(c.FormValue("heat_window")); err == nil && v >= 100 && v <= 5000 {
			s.HeatWindowMS = v
		}
		t1, e1 := strconv.Atoi(c.FormValue("heat_t1"))
		t2, e2 := strconv.Atoi(c.FormValue("heat_t2"))
		t3, e3 := strconv.Atoi(c.FormValue("heat_t3"))
		if e1 == nil && e2 == nil && e3 == nil && t1 > 0 && t1 < t2 && t2 < t3 {
			s.HeatThreshold1 = t1
			s.HeatThreshold2 = t2
			s.HeatThreshold3 = t3
		}
		if v := c.FormValue("heat_color1"); isHexColor(v) {
			s.HeatColor1 = v
		}
		if v := c.FormValue("heat_color2"); isHexColor(v) {
			s.HeatColor2 = v
		}
		if v := c.FormValue("heat_color3"); isHexColor(v) {
			s.HeatColor3 = v
		}
		if v := c.FormValue("heat_base"); isHexColor(v) {
			s.HeatBaseColor = v
		}
	})
	r.publishStats()
	return c.NoContent(http.StatusNoContent)
}

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func isHexColor(s string) bool {
	return hexColorRe.MatchString(s)
}

func (r *tavernHotZoneRoutes) handlePause(c echo.Context) error {
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

func (r *tavernHotZoneRoutes) handleReset(c echo.Context) error {
	r.lab.ResetStats()
	r.lab.RecordActivity("stats reset")
	r.publishStats()
	r.publishActivity()
	return c.NoContent(http.StatusNoContent)
}

func (r *tavernHotZoneRoutes) handleCommand(c echo.Context) error {
	regionID, err := strconv.Atoi(c.FormValue("region"))
	if err != nil {
		regionID, err = strconv.Atoi(c.QueryParam("region"))
	}
	if err != nil || regionID < 1 || regionID > 64 {
		return c.String(http.StatusBadRequest, "invalid region")
	}
	mode := demo.HotZoneMode(c.FormValue("mode"))
	if mode == "" {
		mode = demo.HotZoneMode(c.QueryParam("mode"))
	}
	if mode != demo.HotZoneModeHXPost && mode != demo.HotZoneModeTavern {
		mode = demo.HotZoneModeTavern
	}

	locked := r.lab.ToggleLock(regionID)
	action := "unlocked"
	if locked {
		action = "locked"
	}

	r.lab.RecordReceived(mode)
	r.lab.RecordActivity(fmt.Sprintf("%s region %d via %s", action, regionID, mode))
	r.publishRegion(regionID)
	r.publishStats()
	r.publishActivity()
	return c.NoContent(http.StatusNoContent)
}

func (r *tavernHotZoneRoutes) handleLifecycle(c echo.Context) error {
	action := c.FormValue("action")
	if action == "" {
		action = c.QueryParam("action")
	}
	mode := demo.HotZoneMode(c.FormValue("mode"))
	if mode == "" {
		mode = demo.HotZoneMode(c.QueryParam("mode"))
	}
	if mode != demo.HotZoneModeHXPost && mode != demo.HotZoneModeTavern {
		return c.NoContent(http.StatusBadRequest)
	}
	switch action {
	case "dispatched", "succeeded", "failed":
		r.lab.RecordLifecycle(mode, action)
	default:
		return c.NoContent(http.StatusBadRequest)
	}
	r.publishStats()
	return c.NoContent(http.StatusNoContent)
}

// --- publishers ---

func (r *tavernHotZoneRoutes) publishRegion(id int) {
	r.broker.Publish(topicHZRegion(id), r.renderRegionFrame(id))
}

func (r *tavernHotZoneRoutes) publishStats() {
	r.broker.Publish(topicHZStats, r.renderStatsFrame())
}

func (r *tavernHotZoneRoutes) publishActivity() {
	r.broker.Publish(topicHZActivity, r.renderActivityFrame())
}

// --- renderers ---

func (r *tavernHotZoneRoutes) renderRegionFrame(id int) string {
	region := r.lab.Region(id)
	settings := r.lab.Settings()
	return tavern.NewSSEMessage(fmt.Sprintf("hz-region-%d", id),
		renderToString("hz region", views.HotZoneRegionContent(region, settings)),
	).String()
}

func (r *tavernHotZoneRoutes) renderStatsFrame() string {
	settings := r.lab.Settings()
	stats := r.lab.CommandStats()
	return tavern.NewSSEMessage("hz-stats",
		renderToString("hz stats", views.HotZoneStats(settings, stats, r.lab.Paused())),
	).String()
}

func (r *tavernHotZoneRoutes) renderActivityFrame() string {
	return tavern.NewSSEMessage("hz-activity",
		renderToString("hz activity", views.HotZoneActivityLog(r.lab.Activity())),
	).String()
}

// --- view-model ---

func (r *tavernHotZoneRoutes) buildPageData() views.HotZoneLabData {
	return views.HotZoneLabData{
		Settings: r.lab.Settings(),
		Regions:  r.lab.Regions(),
		Stats:    r.lab.CommandStats(),
		Activity: r.lab.Activity(),
		Paused:   r.lab.Paused(),
	}
}

// --- simulator ---

func (r *tavernHotZoneRoutes) startSimulator(ctx context.Context) {
	for {
		delay := r.lab.JitteredInterval()
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
			if r.lab.Paused() {
				continue
			}
			updated := r.lab.SimTick()
			for _, id := range updated {
				r.publishRegion(id)
			}
			r.publishStats()
		}
	}
}
