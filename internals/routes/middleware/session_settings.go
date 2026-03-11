// setup:feature:session_settings

package middleware

import (
	"net/http"
	"time"

	"catgoose/harmony/internals/domain"
	log "catgoose/harmony/internals/logger"
	"catgoose/harmony/internals/repository"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	sessionUUIDCookie  = "session_uuid"
	settingsContextKey = "sessionSettings"
)

// SessionSettingsMiddleware reads (or creates) a session UUID cookie,
// loads settings from the repository, and stores them on the echo context.
func SessionSettingsMiddleware(repo repository.SessionSettingsRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sessionUUID, isNew := getOrCreateSessionUUID(c)

			settings, err := repo.GetByUUID(c.Request().Context(), sessionUUID)
			if err != nil {
				log.WithContext(c.Request().Context()).Error("Failed to load session settings", "error", err)
				settings = domain.NewDefaultSettings(sessionUUID)
			}
			if settings == nil {
				settings = domain.NewDefaultSettings(sessionUUID)
				if err := repo.Upsert(c.Request().Context(), settings); err != nil {
					log.WithContext(c.Request().Context()).Error("Failed to create session settings", "error", err)
				}
			}

			// Touch if last update was more than 24 hours ago to keep the row fresh.
			if !isNew && time.Since(settings.UpdatedAt) > 24*time.Hour {
				_ = repo.Touch(c.Request().Context(), sessionUUID)
			}

			c.Set(settingsContextKey, settings)
			return next(c)
		}
	}
}

// getOrCreateSessionUUID reads the session UUID cookie or creates a new one.
func getOrCreateSessionUUID(c echo.Context) (string, bool) {
	if cookie, err := c.Cookie(sessionUUIDCookie); err == nil && cookie.Value != "" {
		return cookie.Value, false
	}
	id := uuid.New().String()
	c.SetCookie(&http.Cookie{
		Name:     sessionUUIDCookie,
		Value:    id,
		Path:     "/",
		MaxAge:   86400 * 90, // 90 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return id, true
}

// GetSessionSettings returns the session settings from the echo context.
func GetSessionSettings(c echo.Context) *domain.SessionSettings {
	if s, ok := c.Get(settingsContextKey).(*domain.SessionSettings); ok {
		return s
	}
	return domain.NewDefaultSettings("")
}
