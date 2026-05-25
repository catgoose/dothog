// setup:feature:session_settings

package routes

import (
	"net/http"

	"catgoose/dothog/internal/htmxutil"
	"catgoose/dothog/internal/logger"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/session"
	"catgoose/dothog/web/views"

	// setup:feature:sse:start
	"fmt"
	"sync/atomic"
	"time"

	"github.com/catgoose/tavern"
	// setup:feature:sse:end

	"github.com/labstack/echo/v4"
)

// isValidTheme reports whether theme matches one of the views.DaisyThemes
// the picker exposes. handleTheme uses it for the user self-edit fallback;
// derived apps with an admin-row edit handler can reuse the same canonical
// validation.
func isValidTheme(theme string) bool {
	for _, t := range views.DaisyThemes {
		if t == theme {
			return true
		}
	}
	return false
}

// setup:feature:sse:start

var themeCounter atomic.Int64

// publishThemeChange fans the supplied theme to the cookie-derived session
// topic via the SSE broker so only the session whose preference changed
// receives the event. No-op when broker or sessionUUID is empty, so the
// stale-cookie / pre-cookie cases do not emit ghost events.
func publishThemeChange(broker *tavern.SSEBroker, sessionUUID, theme string) {
	if broker == nil || sessionUUID == "" {
		return
	}
	eventID := fmt.Sprintf("tc%d", themeCounter.Add(1))
	msg := tavern.NewSSEMessage("theme-change", theme).
		WithID(eventID).
		String()
	broker.PublishWithID(ThemeTopicForSession(sessionUUID), eventID, msg)
}

// setup:feature:sse:end

// initThemeRoutes registers the theme picker fragment plus POST /settings/theme
// and POST /settings/layout.
// Both persist into the session_settings row. When SSE is enabled, the theme
// receive path is wired separately by initThemeSSE.
func (ar *AppRoutes) initThemeRoutes() {
	ar.e.GET("/settings/theme/picker", ar.handleThemePicker())
	ar.e.POST("/settings/theme", ar.handleTheme())
	ar.e.POST("/settings/layout", ar.handleLayout())
}

// setup:feature:sse:start

// initThemeSSE wires /sse/theme to the cookie-backed session's private topic
// while preserving the public SSE event name "theme-change". The handler
// derives the session UUID from session middleware, subscribes that
// connection to ThemeTopicForSession(uuid), and streams via tavern.StreamSSE
// so the SSE event field stays "theme-change" (using DynamicGroupHandler
// would leak the private topic name into the event field and break clients
// that listen for "theme-change"). Only called when the sse feature is
// enabled and a broker has been built.
func (ar *AppRoutes) initThemeSSE(broker *tavern.SSEBroker) {
	ar.e.GET("/sse/theme", func(c echo.Context) error {
		s := session.GetSettings(c.Request())
		if s == nil || s.SessionUUID == "" {
			return c.NoContent(http.StatusNoContent)
		}

		c.Response().Header().Set("X-Accel-Buffering", "no")
		lastID := c.Request().Header.Get("Last-Event-ID")
		msgs, unsub := broker.SubscribeFromID(ThemeTopicForSession(s.SessionUUID), lastID)
		defer unsub()

		return tavern.StreamSSE(
			c.Request().Context(),
			c.Response(),
			msgs,
			func(msg string) string { return msg },
			tavern.WithStreamHeartbeat(30*time.Second),
		)
	})
}

// setup:feature:sse:end

// handleThemePicker returns the canonical picker fragment for the current
// session theme. The settings page and the picker's component refresh both
// render through this fragment.
func (ar *AppRoutes) handleThemePicker() echo.HandlerFunc {
	return func(c echo.Context) error {
		return handler.RenderComponent(c, views.ThemePicker(session.GetSettings(c.Request()).Theme))
	}
}

// handleTheme persists the requested theme on the session_settings row.
// In sse builds the send path is this POST and the canonical receive path is
// the per-session theme-change SSE event.
func (ar *AppRoutes) handleTheme() echo.HandlerFunc {
	return func(c echo.Context) error {
		theme := c.FormValue("theme")
		if !isValidTheme(theme) {
			theme = "light"
		}
		settings := session.GetSettings(c.Request())
		settings.Theme = theme
		if ar.deps.SessionStore != nil {
			if err := ar.deps.SessionStore.Upsert(c.Request().Context(), settings); err != nil {
				logger.WithContext(c.Request().Context()).Error("Failed to save theme setting", "error", err)
			}
		}

		// setup:feature:sse:start
		publishThemeChange(ar.broker, settings.SessionUUID, theme)
		// setup:feature:sse:end

		if htmxutil.IsHTMX(c.Request()) && !htmxutil.IsBoosted(c.Request()) {
			// setup:feature:sse:start
			if ar.broker != nil {
				_ = htmxutil.New().
					ReswapNone().
					Write(c.Response())
				return c.NoContent(http.StatusNoContent)
			}
			// setup:feature:sse:end

			_ = htmxutil.New().
				TriggerDetail("app:theme-change", map[string]any{
					"theme":  theme,
					"source": "server",
				}).
				Write(c.Response())
			return handler.RenderComponent(c, views.ThemePicker(theme))
		}

		return c.Redirect(http.StatusSeeOther, "/settings")
	}
}

// handleLayout persists the requested layout and asks the browser to reload.
// HX-Refresh (not HX-Redirect) so the current URL reloads under the new layout.
// 200 (not 204) because HTMX 2.0 sets swap:false for 204, which can drop
// the response headers and skip the refresh.
func (ar *AppRoutes) handleLayout() echo.HandlerFunc {
	return func(c echo.Context) error {
		layout := c.FormValue("layout")
		if layout != session.LayoutApp {
			layout = session.DefaultLayout
		}
		settings := session.GetSettings(c.Request())
		settings.Layout = layout
		if ar.deps.SessionStore != nil {
			if err := ar.deps.SessionStore.Upsert(c.Request().Context(), settings); err != nil {
				logger.WithContext(c.Request().Context()).Error("Failed to save layout setting", "error", err)
			}
		}
		_ = htmxutil.New().Refresh().Write(c.Response())
		return c.String(http.StatusOK, "")
	}
}
