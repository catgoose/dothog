// setup:feature:demo

package routes

import (
	"sort"
	"time"

	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/ssebroker"
	"catgoose/dothog/internal/version"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

func (ar *appRoutes) initAdminSettingsRoutes(broker *ssebroker.SSEBroker) {
	ar.e.GET("/admin/settings", ar.handleAdminSettings(broker))
	ar.e.GET("/admin/settings/system", ar.handleAdminSettingsSystem)
	ar.e.GET("/admin/settings/sse", ar.handleAdminSettingsSSE(broker))
}

func (ar *appRoutes) handleAdminSettings(broker *ssebroker.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := ar.buildAdminPanelData(broker, c)
		return handler.RenderBaseLayout(c, views.AdminSettingsPage(data))
	}
}

func (ar *appRoutes) handleAdminSettingsSystem(c echo.Context) error {
	stats := ssebroker.CollectRuntimeStats(ar.startTime)
	return handler.RenderComponent(c, views.AdminSettingsSystemFragment(stats, formatUptime(time.Since(ar.startTime))))
}

func (ar *appRoutes) handleAdminSettingsSSE(broker *ssebroker.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		counts := broker.TopicCounts()
		return handler.RenderComponent(c, views.AdminSettingsSSEFragment(counts))
	}
}

func (ar *appRoutes) buildAdminPanelData(broker *ssebroker.SSEBroker, c echo.Context) views.AdminPanelData {
	stats := ssebroker.CollectRuntimeStats(ar.startTime)
	counts := broker.TopicCounts()

	// Collect registered routes
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

	// Feature flags — detect from registered routes
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

