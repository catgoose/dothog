// setup:feature:session_settings

package routes

import (
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

// initAdminSessionsRoutes registers the framework-owned admin surface for the
// session_settings store: list and delete by UUID. Only ships when the
// session_settings feature is kept.
func (ar *AppRoutes) initAdminSessionsRoutes() {
	ar.e.GET("/admin/sessions", ar.handleSessionsPage)
	ar.e.GET("/admin/sessions/table", ar.handleSessionsTable)
	ar.e.DELETE("/admin/sessions/:uuid", ar.handleSessionsDelete)
}

func (ar *AppRoutes) handleSessionsPage(c echo.Context) error {
	if ar.repos.SessionSettings == nil {
		return handler.HandleHypermediaError(c, 500, "Session settings not configured", nil)
	}
	sessions, err := ar.repos.SessionSettings.ListAll(c.Request().Context())
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load sessions", err)
	}
	return handler.RenderBaseLayout(c, views.AdminSessionsPage(sessions))
}

func (ar *AppRoutes) handleSessionsTable(c echo.Context) error {
	if ar.repos.SessionSettings == nil {
		return handler.HandleHypermediaError(c, 500, "Session settings not configured", nil)
	}
	sessions, err := ar.repos.SessionSettings.ListAll(c.Request().Context())
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load sessions", err)
	}
	return handler.RenderComponent(c, views.AdminSessionsTable(sessions))
}

// handleSessionsDelete drops a session_settings row and returns the refreshed
// table so HTMX can swap it in place. The DELETE-By-UUID path is idempotent,
// so duplicate clicks and stale fragments don't 4xx.
func (ar *AppRoutes) handleSessionsDelete(c echo.Context) error {
	if ar.repos.SessionSettings == nil {
		return handler.HandleHypermediaError(c, 500, "Session settings not configured", nil)
	}
	uuid := c.Param("uuid")
	if uuid == "" {
		return handler.HandleHypermediaError(c, 400, "Missing session uuid", nil)
	}
	if err := ar.repos.SessionSettings.DeleteByUUID(c.Request().Context(), uuid); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to delete session", err)
	}
	sessions, err := ar.repos.SessionSettings.ListAll(c.Request().Context())
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to reload sessions", err)
	}
	return handler.RenderComponent(c, views.AdminSessionsTable(sessions))
}
