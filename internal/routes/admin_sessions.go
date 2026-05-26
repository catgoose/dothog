// setup:feature:session_settings

package routes

import (
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"
	"net/http"
	"strings"

	// setup:feature:sse:start
	"catgoose/dothog/internal/session"
	// setup:feature:sse:end
	"github.com/labstack/echo/v4"
)

// initAdminSessionsRoutes registers the framework-owned admin surface for the
// session_settings store: list, delete by UUID, and batch-delete by UUID
// list. Only ships when the session_settings feature is kept.
func (ar *AppRoutes) initAdminSessionsRoutes() {
	sessions := ar.e.Group("/admin/sessions")
	sessions.GET("", ar.handleSessionsPage)
	sessions.GET("/table", ar.handleSessionsTable)
	sessions.POST("/delete", ar.handleSessionsBatchDelete)
	sessions.POST("/:uuid/theme", ar.handleSessionsUpdateTheme)
	sessions.DELETE("/:uuid", ar.handleSessionsDelete)
}

func (ar *AppRoutes) handleSessionsPage(c echo.Context) error {
	if ar.deps.SessionSettings == nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Session settings not configured", nil)
	}
	sessions, err := ar.deps.SessionSettings.ListAll(c.Request().Context())
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Failed to load sessions", err)
	}
	return handler.RenderBaseLayout(c, views.AdminSessionsPage(sessions))
}

func (ar *AppRoutes) handleSessionsTable(c echo.Context) error {
	if ar.deps.SessionSettings == nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Session settings not configured", nil)
	}
	sessions, err := ar.deps.SessionSettings.ListAll(c.Request().Context())
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Failed to load sessions", err)
	}
	return handler.RenderComponent(c, views.AdminSessionsTable(sessions))
}

// handleSessionsDelete drops a session_settings row and returns the refreshed
// table so HTMX can swap it in place. The DELETE-By-UUID path is idempotent,
// so duplicate clicks and stale fragments don't 4xx.
func (ar *AppRoutes) handleSessionsDelete(c echo.Context) error {
	if ar.deps.SessionSettings == nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Session settings not configured", nil)
	}
	uuid := c.Param("uuid")
	if uuid == "" {
		return handler.HandleHypermediaError(c, http.StatusBadRequest, "Missing session uuid", nil)
	}
	if err := ar.deleteSessionSettingsRow(c, uuid); err != nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Failed to delete session", err)
	}
	sessions, err := ar.deps.SessionSettings.ListAll(c.Request().Context())
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Failed to reload sessions", err)
	}
	return handler.RenderComponent(c, views.AdminSessionsTable(sessions))
}

// handleSessionsBatchDelete drops every session_settings row in the
// comma-separated "uuids" form field and returns the refreshed table. Each
// per-UUID delete is idempotent, so duplicates and stale uuids no-op rather
// than failing the batch.
func (ar *AppRoutes) handleSessionsBatchDelete(c echo.Context) error {
	if ar.deps.SessionSettings == nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Session settings not configured", nil)
	}
	raw := c.FormValue("uuids")
	if raw != "" {
		for _, uuid := range strings.Split(raw, ",") {
			uuid = strings.TrimSpace(uuid)
			if uuid == "" {
				continue
			}
			if err := ar.deleteSessionSettingsRow(c, uuid); err != nil {
				return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Failed to delete session", err)
			}
		}
	}
	sessions, err := ar.deps.SessionSettings.ListAll(c.Request().Context())
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Failed to reload sessions", err)
	}
	return handler.RenderComponent(c, views.AdminSessionsTable(sessions))
}

func (ar *AppRoutes) deleteSessionSettingsRow(c echo.Context, uuid string) error {
	existing, err := ar.deps.SessionSettings.GetByUUID(c.Request().Context(), uuid)
	if err != nil {
		return err
	}
	if err := ar.deps.SessionSettings.DeleteByUUID(c.Request().Context(), uuid); err != nil {
		return err
	}
	// setup:feature:sse:start
	if existing != nil {
		publishThemeChange(ar.broker, uuid, session.DefaultTheme)
	}
	// setup:feature:sse:end
	_ = existing
	return nil
}

// handleSessionsUpdateTheme rewrites one session row's Theme and returns the
// refreshed table fragment. When the row exists, the updated theme is also
// published to that session's private theme SSE topic so every tab sharing
// that cookie-backed session converges without leaking the event to other
// sessions.
func (ar *AppRoutes) handleSessionsUpdateTheme(c echo.Context) error {
	if ar.deps.SessionSettings == nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Session settings not configured", nil)
	}
	uuid := c.Param("uuid")
	if uuid == "" {
		return handler.HandleHypermediaError(c, http.StatusBadRequest, "Missing session uuid", nil)
	}
	theme := c.FormValue("theme")
	if !isValidTheme(theme) {
		return c.String(http.StatusBadRequest, "Invalid theme")
	}
	updated, err := ar.deps.SessionSettings.UpdateThemeByUUID(c.Request().Context(), uuid, theme)
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Failed to update session theme", err)
	}
	// setup:feature:sse:start
	if updated {
		publishThemeChange(ar.broker, uuid, theme)
	}
	// setup:feature:sse:end
	_ = updated
	sessions, err := ar.deps.SessionSettings.ListAll(c.Request().Context())
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Failed to reload sessions", err)
	}
	return handler.RenderComponent(c, views.AdminSessionsTable(sessions))
}
