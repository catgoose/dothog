// setup:feature:session_settings

package routes

import (
	"net/http"

	"catgoose/dothog/internal/logger"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/session"
	"catgoose/dothog/web/views"

	// setup:feature:sse:start
	"fmt"
	"sync/atomic"

	"github.com/catgoose/tavern"
	// setup:feature:sse:end

	"github.com/labstack/echo/v4"
)

// setup:feature:sse:start
var themeCounter atomic.Int64

// setup:feature:sse:end

// initThemeRoutes registers POST /settings/theme and POST /settings/layout.
// Both persist into the session_settings row. Cross-browser broadcast is wired
// separately by initThemeSSE when the sse feature is enabled.
func (ar *appRoutes) initThemeRoutes() {
	ar.e.POST("/settings/theme", ar.handleTheme())
	ar.e.POST("/settings/layout", ar.handleLayout())
}

// setup:feature:sse:start

// initThemeSSE adds the cross-browser theme-change feed and replay policy.
// Only called when the sse feature is enabled and a broker has been built.
func (ar *appRoutes) initThemeSSE(broker *tavern.SSEBroker) {
	broker.SetReplayPolicy(TopicThemeChange, 1)
	broker.SetReplayGapPolicy(TopicThemeChange, tavern.GapFallbackToSnapshot, nil)
	ar.e.GET("/sse/theme", echo.WrapHandler(broker.SSEHandler(TopicThemeChange)))
}

// setup:feature:sse:end

// handleTheme persists the requested theme on the session_settings row.
// In sse builds the change is also broadcast so other tabs/devices receive it.
func (ar *appRoutes) handleTheme() echo.HandlerFunc {
	return func(c echo.Context) error {
		theme := c.FormValue("theme")
		valid := false
		for _, t := range views.DaisyThemes {
			if t == theme {
				valid = true
				break
			}
		}
		if !valid {
			theme = "light"
		}
		settings := session.GetSettings(c.Request())
		settings.Theme = theme
		if ar.repos.Settings != nil {
			if err := ar.repos.Settings.Upsert(c.Request().Context(), settings); err != nil {
				logger.WithContext(c.Request().Context()).Error("Failed to save theme setting", "error", err)
			}
		}

		// setup:feature:sse:start
		if ar.broker != nil {
			eventID := fmt.Sprintf("tc%d", themeCounter.Add(1))
			msg := tavern.NewSSEMessage("theme-change", theme).
				WithID(eventID).
				String()
			ar.broker.PublishWithID(TopicThemeChange, eventID, msg)
		}
		// setup:feature:sse:end

		return handler.RenderComponent(c, views.ThemeChanged(theme))
	}
}

// handleLayout persists the requested layout and asks the browser to reload.
// HX-Refresh (not HX-Redirect) so the current URL reloads under the new layout.
// 200 (not 204) because HTMX 2.0 sets swap:false for 204, which can drop
// the response headers and skip the refresh.
func (ar *appRoutes) handleLayout() echo.HandlerFunc {
	return func(c echo.Context) error {
		layout := c.FormValue("layout")
		if layout != session.LayoutApp {
			layout = session.DefaultLayout
		}
		settings := session.GetSettings(c.Request())
		settings.Layout = layout
		if ar.repos.Settings != nil {
			if err := ar.repos.Settings.Upsert(c.Request().Context(), settings); err != nil {
				logger.WithContext(c.Request().Context()).Error("Failed to save layout setting", "error", err)
			}
		}
		c.Response().Header().Set("HX-Refresh", "true")
		return c.String(http.StatusOK, "")
	}
}
