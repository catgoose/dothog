// setup:feature:demo

package middleware

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"

	"catgoose/dothog/internal/routes/hypermedia"

	"github.com/labstack/echo/v4"
)

// VisitRecorder records page visits for frecency tracking.
type VisitRecorder interface {
	RecordVisit(ctx context.Context, sessionID, path, title string) error
}

// FrecencyMiddleware sets the session ID on the request context and records
// page visits for full-page GET requests to valid routes.
func FrecencyMiddleware(recorder VisitRecorder, validRoutes map[string]bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sessionID := getSessionID(c)

			// Set session ID on the Go context for downstream LinkSources.
			ctx := hypermedia.WithSessionID(c.Request().Context(), sessionID)
			c.SetRequest(c.Request().WithContext(ctx))

			// Only record visits for full-page GET navigations to valid routes.
			if c.Request().Method == http.MethodGet &&
				c.Request().Header.Get("HX-Request") != "true" &&
				validRoutes[c.Request().URL.Path] {

				path := c.Request().URL.Path
				title := hypermedia.TitleFromPath(path)
				go func() {
					_ = recorder.RecordVisit(context.Background(), sessionID, path, title)
				}()
			}

			return next(c)
		}
	}
}

// getSessionID returns a stable session identifier. It tries a cookie first,
// then falls back to a hash of IP + User-Agent.
func getSessionID(c echo.Context) string {
	if cookie, err := c.Cookie("dothog_sid"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	h := sha256.Sum256([]byte(c.RealIP() + "|" + c.Request().UserAgent()))
	return fmt.Sprintf("%x", h[:8])
}
