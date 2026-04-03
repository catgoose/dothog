// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"catgoose/dothog/internal/health"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/version"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

// ── ScheduledPublisher for admin panel ─────────────────────────────────────

var adminPub *tavern.ScheduledPublisher

// ── Routes ──────────────────────────────────────────────────────────────────

func (ar *appRoutes) initAdminSettingsRoutes(broker *tavern.SSEBroker) {
	ar.e.GET("/admin/settings", ar.handleAdminSettings(broker))
	ar.e.POST("/admin/settings/interval", handleAdminInterval)
	ar.e.GET("/sse/admin", handleSSEAdmin(broker))

	adminPub = broker.NewScheduledPublisher(TopicAdminPanel, tavern.WithBaseTick(500*time.Millisecond))
	adminPub.Register("system-metrics", 5*time.Second, func(ctx context.Context, buf *bytes.Buffer) error {
		stats := health.CollectRuntimeStats(ar.startTime)
		uptime := formatUptime(time.Since(ar.startTime))
		return views.OOBAdminSystemMetrics(stats, uptime).Render(ctx, buf)
	})
	adminPub.Register("sse-counts", 3*time.Second, func(ctx context.Context, buf *bytes.Buffer) error {
		counts := broker.TopicCounts()
		return views.OOBAdminSSECounts(counts).Render(ctx, buf)
	})
	adminPub.Register("health", 5*time.Second, func(ctx context.Context, buf *bytes.Buffer) error {
		h := health.Check(ctx, ar.healthCfg)
		return views.OOBAdminHealth(h).Render(ctx, buf)
	})
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
	} else if ms > 600000 {
		ms = 600000
	}
	adminPub.SetInterval(section, time.Duration(ms)*time.Millisecond)
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
				fmt.Fprint(c.Response(), tavern.NewSSEMessage("admin-panel", msg).String())
				flusher.Flush()
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
