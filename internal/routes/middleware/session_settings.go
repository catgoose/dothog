// setup:feature:session_settings

package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"catgoose/dothog/internal/domain"
	"catgoose/dothog/internal/logger"

	"github.com/labstack/echo/v4"
)

// SessionSettingsProvider is the subset of session-settings operations that
// the middleware needs: look up, create-or-update, and touch a row.
type SessionSettingsProvider interface {
	GetByUUID(ctx context.Context, uuid string) (*domain.SessionSettings, error)
	Upsert(ctx context.Context, s *domain.SessionSettings) error
	Touch(ctx context.Context, uuid string) error
}

// SessionIDFunc returns the session identifier for the current request.
// When nil, the middleware uses a shared UUID (demo) or falls back to
// a random cookie-based session ID (derived apps).
type SessionIDFunc func(c echo.Context) string

const (
	settingsContextKey = "sessionSettings"
	sessionCookieName  = "{{BINARY_NAME}}_session_id"
	// setup:feature:demo:start
	// sharedSessionUUID is used for all visitors so the demo behaves as a
	// single-user application — every browser reads/writes the same row.
	sharedSessionUUID = "00000000-0000-0000-0000-000000000000"
	// setup:feature:demo:end
)

// SessionSettingsMiddleware loads per-session settings and stores them on the
// echo context. The session ID comes from idFunc (e.g. Crooner's SCS token).
// When idFunc is nil, the demo uses a shared UUID; derived apps fall back to
// a random cookie-based session ID.
func SessionSettingsMiddleware(repo SessionSettingsProvider, idFunc SessionIDFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			sessionID := ""
			if idFunc != nil {
				sessionID = idFunc(c)
			}
			if sessionID == "" {
				// setup:feature:demo:start
				sessionID = sharedSessionUUID
				// setup:feature:demo:end
				// setup:feature:!demo — when demo is stripped, the line above
				// is removed, leaving sessionID empty. The fallback below
				// generates a random cookie-based session ID.
			}
			if sessionID == "" {
				sessionID = getOrCreateSessionCookie(c)
			}

			settings, err := repo.GetByUUID(ctx, sessionID)
			if err != nil {
				logger.WithContext(ctx).Error("Failed to load session settings", "error", err)
				settings = domain.NewDefaultSettings(sessionID)
			}
			if settings == nil {
				settings = domain.NewDefaultSettings(sessionID)
				if err := repo.Upsert(ctx, settings); err != nil {
					logger.WithContext(ctx).Error("Failed to create session settings", "error", err)
				}
			}

			if time.Since(settings.UpdatedAt) > 24*time.Hour {
				_ = repo.Touch(ctx, sessionID)
			}

			c.Set(settingsContextKey, settings)
			return next(c)
		}
	}
}

// GetSessionSettings returns the session settings from the echo context.
func GetSessionSettings(c echo.Context) *domain.SessionSettings {
	if s, ok := c.Get(settingsContextKey).(*domain.SessionSettings); ok {
		return s
	}
	return domain.NewDefaultSettings("")
}

// getOrCreateSessionCookie reads the session cookie or creates a new random one.
func getOrCreateSessionCookie(c echo.Context) string {
	if cookie, err := c.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	id := randomUUID()
	c.SetCookie(&http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return id
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
