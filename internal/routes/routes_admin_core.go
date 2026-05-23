package routes

import (
	"catgoose/dothog/internal/health"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

// initAdminCoreRoutes registers the scaffold-facing admin pages every derived
// app inherits: the admin index, the structured health page and HTMX
// fragment, and the debug page. Demo-only admin pages (/admin/system,
// /admin/config, …) live in routes_admin_demo.go; session-settings admin
// (DELETE/list) lives in routes_admin_sessions.go.
func (ar *AppRoutes) initAdminCoreRoutes() {
	ar.e.GET("/admin", handler.HandleComponent(views.AdminIndexPage()))
	ar.e.GET("/admin/health", ar.handleAdminHealth)
	ar.e.GET("/admin/health/check", ar.handleAdminHealthCheck)
	ar.e.GET("/admin/debug", handler.HandleComponent(views.AdminDebugPage()))
}

// healthIntervalsFn returns the current admin interval snapshot. Defaults to
// a no-op nil-returning function and is overridden in init() inside the
// demo-gated routes_admin_settings.go when the demo feature is present.
var healthIntervalsFn = func() map[string]int { return nil }

func (ar *AppRoutes) handleAdminHealth(c echo.Context) error {
	h := health.Check(c.Request().Context(), ar.healthCfg)
	return handler.RenderBaseLayout(c, views.AdminHealthPage(h, healthIntervalsFn()))
}

func (ar *AppRoutes) handleAdminHealthCheck(c echo.Context) error {
	h := health.Check(c.Request().Context(), ar.healthCfg)
	return handler.RenderComponent(c, views.AdminHealthFragment(h))
}
