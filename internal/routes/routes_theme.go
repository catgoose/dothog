// setup:feature:session_settings

package routes

import (
	"net/http"

	"catgoose/dothog/internal/logger"
	// setup:feature:session_settings:start
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"
	// setup:feature:session_settings:end
	"github.com/catgoose/tavern"

	// setup:feature:session_settings:start
	"catgoose/dothog/internal/session"
	// setup:feature:session_settings:end
	"github.com/labstack/echo/v4"
)

// setup:feature:session_settings:start

func (ar *appRoutes) initThemeRoutes(broker *tavern.SSEBroker) {
	ar.e.POST("/settings/theme", ar.handleTheme(broker))
	ar.e.POST("/settings/layout", ar.handleLayout())
	ar.e.GET("/sse/theme", echo.WrapHandler(broker.SSEHandler(TopicThemeChange)))
}

// handleTheme updates the shared theme setting and broadcasts to all browsers.
func (ar *appRoutes) handleTheme(broker *tavern.SSEBroker) echo.HandlerFunc {
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

		// Broadcast theme change to all connected browsers.
		if broker.HasSubscribers(TopicThemeChange) {
			msg := tavern.NewSSEMessage("theme-change", theme).String()
			broker.Publish(TopicThemeChange, msg)
		}

		return handler.RenderComponent(c, views.ThemeChanged(theme))
	}
}

// handleLayout updates the shared layout setting and refreshes the page.
// Uses HX-Refresh instead of HX-Redirect so the browser reloads the current
// page with the new layout regardless of which page the toggle lives on.
// Returns 200 (not 204) because HTMX 2.0 responseHandling sets swap:false for
// 204, which can prevent response headers from being processed reliably.
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

// setup:feature:session_settings:end

