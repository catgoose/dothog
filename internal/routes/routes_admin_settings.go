// setup:feature:demo

package routes

import (
	"bytes"
	"context"
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
	"system-metrics": 5000, // 5s
	"sse-counts":     3000, // 3s
	"health":         5000, // 5s
}

var (
	adminIntervals struct {
		intervals map[string]int
		mu        sync.RWMutex
	}
	adminPub *tavern.ScheduledPublisher
)

func initAdminIntervals() {
	adminIntervals.intervals = make(map[string]int, len(adminDefaultIntervals))
	for id, iv := range adminDefaultIntervals {
		adminIntervals.intervals[id] = iv
	}
}

// ── Routes ──────────────────────────────────────────────────────────────────

func (ar *appRoutes) initAdminSettingsRoutes(broker *tavern.SSEBroker) {
	initAdminIntervals()
	ar.e.GET("/admin/settings", ar.handleAdminSettings(broker))
	ar.e.POST("/admin/settings/interval", handleAdminInterval)
	ar.e.GET("/sse/admin", echo.WrapHandler(broker.SSEHandler(TopicAdminPanel)))

	adminPub = ar.newAdminPublisher(broker)
	broker.RunPublisher(ar.ctx, adminPub.Start)
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

	if adminPub != nil {
		adminPub.SetInterval(section, time.Duration(ms)*time.Millisecond)
	}

	return c.NoContent(http.StatusNoContent)
}

// ── Publisher ────────────────────────────────────────────────────────────────

func (ar *appRoutes) newAdminPublisher(broker *tavern.SSEBroker) *tavern.ScheduledPublisher {
	pub := broker.NewScheduledPublisher(TopicAdminPanel, tavern.WithBaseTick(500*time.Millisecond))

	pub.Register("system-metrics", time.Duration(adminDefaultIntervals["system-metrics"])*time.Millisecond,
		func(ctx context.Context, buf *bytes.Buffer) error {
			stats := health.CollectRuntimeStats(ar.startTime)
			uptime := formatUptime(time.Since(ar.startTime))
			return views.OOBAdminSystemMetrics(stats, uptime).Render(ctx, buf)
		})

	pub.Register("sse-counts", time.Duration(adminDefaultIntervals["sse-counts"])*time.Millisecond,
		func(ctx context.Context, buf *bytes.Buffer) error {
			counts := broker.TopicCounts()
			return views.OOBAdminSSECounts(counts).Render(ctx, buf)
		})

	pub.Register("health", time.Duration(adminDefaultIntervals["health"])*time.Millisecond,
		func(ctx context.Context, buf *bytes.Buffer) error {
			h := health.Check(ctx, ar.healthCfg)
			return views.OOBAdminHealth(h).Render(ctx, buf)
		})

	return pub
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
