// setup:feature:sse

package routes

import (
	"fmt"
	"net/http"

	"catgoose/dothog/internal/logger"
	// setup:feature:session_settings:start
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"
	// setup:feature:session_settings:end
	"catgoose/dothog/internal/ssetopics"
	"github.com/catgoose/tavern"

	// setup:feature:session_settings:start
	"github.com/catgoose/porter"
	// setup:feature:session_settings:end
	"github.com/labstack/echo/v4"
)

// setup:feature:session_settings:start

func (ar *appRoutes) initThemeRoutes(broker *tavern.SSEBroker) {
	ar.e.POST("/settings/theme", ar.handleTheme(broker))
	ar.e.POST("/settings/layout", ar.handleLayout())
	ar.e.GET("/sse/theme", handleSSETheme(broker))
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
		settings := porter.GetSessionSettings(c)
		settings.Theme = theme
		if ar.settingsRepo != nil {
			if err := ar.settingsRepo.Upsert(c.Request().Context(), settings); err != nil {
				logger.WithContext(c.Request().Context()).Error("Failed to save theme setting", "error", err)
			}
		}

		// Broadcast theme change to all connected browsers.
		if broker.HasSubscribers(ssetopics.TopicThemeChange) {
			msg := tavern.NewSSEMessage("theme-change", theme).String()
			broker.Publish(ssetopics.TopicThemeChange, msg)
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
		if layout != porter.LayoutApp {
			layout = porter.DefaultLayout
		}
		settings := porter.GetSessionSettings(c)
		settings.Layout = layout
		if ar.settingsRepo != nil {
			if err := ar.settingsRepo.Upsert(c.Request().Context(), settings); err != nil {
				logger.WithContext(c.Request().Context()).Error("Failed to save layout setting", "error", err)
			}
		}
		c.Response().Header().Set("HX-Refresh", "true")
		return c.String(http.StatusOK, "")
	}
}

// setup:feature:session_settings:end

// handleSSETheme streams theme-change events to all connected browsers.
func handleSSETheme(broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().WriteHeader(http.StatusOK)

		flusher, ok := c.Response().Writer.(http.Flusher)
		if !ok {
			return fmt.Errorf("streaming unsupported")
		}

		ch, unsub := broker.Subscribe(ssetopics.TopicThemeChange)
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
				fmt.Fprint(c.Response(), msg)
				flusher.Flush()
			}
		}
	}
}
