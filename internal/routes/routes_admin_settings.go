// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"catgoose/dothog/internal/health"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/version"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

// ── Per-section interval state ──────────────────────────────────────────────

var adminDefaultIntervals = map[string]int{
	"system-metrics": 5000,  // 5s
	"sse-counts":     3000,  // 3s
	"health":         5000,  // 5s
}

var adminIntervals struct {
	intervals map[string]int
	lastSent  map[string]time.Time
	mu        sync.RWMutex
}

func initAdminIntervals() {
	adminIntervals.intervals = make(map[string]int, len(adminDefaultIntervals))
	adminIntervals.lastSent = make(map[string]time.Time, len(adminDefaultIntervals))
	for id, iv := range adminDefaultIntervals {
		adminIntervals.intervals[id] = iv
	}
}

// ── Routes ──────────────────────────────────────────────────────────────────

var adminBufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func (ar *appRoutes) initAdminSettingsRoutes(broker *tavern.SSEBroker) {
	initAdminIntervals()
	ar.e.GET("/admin/settings", ar.handleAdminSettings(broker))
	ar.e.POST("/admin/settings/interval", handleAdminInterval)
	ar.e.GET("/sse/admin", handleSSEAdmin(broker))

	go ar.publishAdminPanel(broker)
}

func (ar *appRoutes) handleAdminSettings(broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := ar.buildAdminPanelData(broker, c)
		return handler.RenderBaseLayout(c, views.AdminSettingsPage(data))
	}
}

func handleAdminInterval(c echo.Context) error {
	section := c.FormValue("section")
	ms, _ := strconv.Atoi(c.FormValue("interval_ms"))
	if ms < 100 {
		ms = 100
	} else if ms > 86400000 {
		ms = 86400000
	}
	adminIntervals.mu.Lock()
	adminIntervals.intervals[section] = ms
	adminIntervals.mu.Unlock()
	return c.NoContent(http.StatusNoContent)
}

func handleSSEAdmin(broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().WriteHeader(http.StatusOK)
		flusher, ok := c.Response().Writer.(http.Flusher)
		if !ok {
			return fmt.Errorf("streaming not supported")
		}
		flusher.Flush()

		ch, unsub := broker.Subscribe(TopicAdminPanel)
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
				fmt.Fprint(c.Response(), msg) //nolint:errcheck // SSE stream; client disconnect handled by context
				flusher.Flush()
			}
		}
	}
}

// ── Publisher ────────────────────────────────────────────────────────────────

func (ar *appRoutes) publishAdminPanel(broker *tavern.SSEBroker) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	ctx := context.Background()

	for {
		select {
		case <-ar.ctx.Done():
			return
		case <-ticker.C:
			if !broker.HasSubscribers(TopicAdminPanel) {
				continue
			}

			now := time.Now()
			buf := adminBufPool.Get().(*bytes.Buffer)
			buf.Reset()
			needsPublish := false

			adminIntervals.mu.Lock()

			// System metrics
			if ms := adminIntervals.intervals["system-metrics"]; ms > 0 {
				if now.Sub(adminIntervals.lastSent["system-metrics"]) >= time.Duration(ms)*time.Millisecond {
					stats := health.CollectRuntimeStats(ar.startTime)
					uptime := formatUptime(time.Since(ar.startTime))
					_ = views.OOBAdminSystemMetrics(stats, uptime).Render(ctx, buf)
					adminIntervals.lastSent["system-metrics"] = now
					needsPublish = true
				}
			}

			// SSE topic counts
			if ms := adminIntervals.intervals["sse-counts"]; ms > 0 {
				if now.Sub(adminIntervals.lastSent["sse-counts"]) >= time.Duration(ms)*time.Millisecond {
					counts := broker.TopicCounts()
					_ = views.OOBAdminSSECounts(counts).Render(ctx, buf)
					adminIntervals.lastSent["sse-counts"] = now
					needsPublish = true
				}
			}

			// Health (for /admin/health page)
			if ms := adminIntervals.intervals["health"]; ms > 0 {
				if now.Sub(adminIntervals.lastSent["health"]) >= time.Duration(ms)*time.Millisecond {
					h := health.Check(ctx, ar.healthCfg)
					_ = views.OOBAdminHealth(h).Render(ctx, buf)
					adminIntervals.lastSent["health"] = now
					needsPublish = true
				}
			}

			adminIntervals.mu.Unlock()

			if needsPublish {
				msg := tavern.NewSSEMessage("admin-panel", buf.String()).String()
				adminBufPool.Put(buf)
				broker.Publish(TopicAdminPanel, msg)
			} else {
				adminBufPool.Put(buf)
			}
		}
	}
}

// ── Data builders ───────────────────────────────────────────────────────────

func (ar *appRoutes) buildAdminPanelData(broker *tavern.SSEBroker, c echo.Context) views.AdminPanelData {
	stats := health.CollectRuntimeStats(ar.startTime)
	counts := broker.TopicCounts()

	var routes []views.RouteInfo
	for _, r := range c.Echo().Routes() {
		if r.Path == "" || r.Path == "/*" {
			continue
		}
		routes = append(routes, views.RouteInfo{Method: r.Method, Path: r.Path})
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})

	features := detectFeatures(c.Echo())

	return views.AdminPanelData{
		AppName:   "dothog",
		Version:   version.Version,
		Uptime:    formatUptime(time.Since(ar.startTime)),
		Status:    "healthy",
		Stats:     stats,
		SSECounts: counts,
		Features:  features,
		Routes:    routes,
	}
}

func detectFeatures(e *echo.Echo) []views.FeatureFlag {
	routeSet := make(map[string]bool)
	for _, r := range e.Routes() {
		routeSet[r.Path] = true
	}

	check := func(name, path string) views.FeatureFlag {
		return views.FeatureFlag{Name: name, Active: routeSet[path]}
	}

	return []views.FeatureFlag{
		check("demo", "/dashboard"),
		check("sse", "/sse/dashboard"),
		check("session_settings", "/user/settings"),
		check("auth", "/auth/callback"),
		check("csrf", "/settings/theme"),
		check("database", "/admin"),
		check("avatar", "/avatar/:email"),
	}
}
