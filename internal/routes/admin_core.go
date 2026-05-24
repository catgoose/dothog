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
// /admin/config, …) live in admin_demo.go; session-settings admin
// (DELETE/list) lives in admin_sessions.go.
func (ar *AppRoutes) initAdminCoreRoutes() {
	admin := ar.e.Group("/admin")
	admin.GET("", handler.HandleComponent(views.AdminIndexPage()))
	admin.GET("/health", ar.handleAdminHealth)
	admin.GET("/health/check", ar.handleAdminHealthCheck)
	admin.GET("/debug", handler.HandleComponent(views.AdminDebugPage()))
	ar.initAdminErrorScenariosRoutes()
}

func (ar *AppRoutes) handleAdminHealth(c echo.Context) error {
	h := health.Check(c.Request().Context(), ar.healthCfg)
	var intervals map[string]int
	if ar.healthIntervals != nil {
		intervals = ar.healthIntervals()
	}
	return handler.RenderBaseLayout(c, views.AdminHealthPage(h, intervals))
}

func (ar *AppRoutes) handleAdminHealthCheck(c echo.Context) error {
	h := health.Check(c.Request().Context(), ar.healthCfg)
	return handler.RenderComponent(c, views.AdminHealthFragment(h))
}
