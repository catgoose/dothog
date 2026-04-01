// setup:feature:session_settings
package session

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type settingsKeyType struct{}

var settingsCtxKey settingsKeyType

// SessionSettings holds per-session user preferences keyed by a browser UUID
// cookie. The struct is designed to be stored in a database row; the db tags
// match the expected column names.
type SessionSettings struct {
	SessionUUID string            `db:"SessionUUID"`
	Theme       string            `db:"Theme"`
	Layout      string            `db:"Layout"`
	Extra       map[string]string `db:"Extra" json:"extra,omitempty"`
	UpdatedAt   time.Time         `db:"UpdatedAt"`
	ID          int               `db:"Id"`
}

// GetExtra returns the value for key, or empty string if not set.
func (s *SessionSettings) GetExtra(key string) string {
	if s.Extra == nil {
		return ""
	}
	return s.Extra[key]
}

// SetExtra sets a key-value pair in Extra, initializing the map if nil.
func (s *SessionSettings) SetExtra(key, value string) {
	if s.Extra == nil {
		s.Extra = make(map[string]string)
	}
	s.Extra[key] = value
}

// MarshalExtra returns the Extra map serialized as a JSON string.
func (s *SessionSettings) MarshalExtra() (string, error) {
	if s.Extra == nil {
		return "{}", nil
	}
	b, err := json.Marshal(s.Extra)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// UnmarshalExtra populates the Extra map from a JSON string.
func (s *SessionSettings) UnmarshalExtra(data string) error {
	if data == "" {
		s.Extra = make(map[string]string)
		return nil
	}
	m := make(map[string]string)
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return err
	}
	s.Extra = m
	return nil
}

const (
	DefaultTheme  = "light"
	DefaultLayout = "classic"
	LayoutApp     = "app"
)

// NewDefaultSettings returns a SessionSettings with defaults for the given UUID.
func NewDefaultSettings(uuid string) *SessionSettings {
	return &SessionSettings{
		SessionUUID: uuid,
		Theme:       DefaultTheme,
		Layout:      DefaultLayout,
		Extra:       make(map[string]string),
	}
}

// SessionConfig holds session middleware configuration.
type SessionConfig struct {
	CookieName string
	Logger     *slog.Logger
}

func (cfg SessionConfig) cookieName() string {
	if cfg.CookieName != "" {
		return cfg.CookieName
	}
	return "session_id"
}

func (cfg SessionConfig) logger() *slog.Logger {
	if cfg.Logger != nil {
		return cfg.Logger
	}
	return slog.Default()
}

// Provider is the interface for session-settings persistence.
type Provider interface {
	GetByUUID(ctx context.Context, uuid string) (*SessionSettings, error)
	Upsert(ctx context.Context, s *SessionSettings) error
	Touch(ctx context.Context, uuid string) error
}

// IDFunc returns the session identifier for the current request.
type IDFunc func(r *http.Request) string

// Middleware returns middleware that loads per-session settings and stores
// them on the request context for downstream handlers.
func Middleware(repo Provider, idFunc IDFunc, cfgs ...SessionConfig) func(http.Handler) http.Handler {
	var cfg SessionConfig
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			sessionID := ""
			if idFunc != nil {
				sessionID = idFunc(r)
			}
			if sessionID == "" {
				sessionID = getOrCreateSessionCookie(w, r, cfg.cookieName())
			}

			settings, err := repo.GetByUUID(ctx, sessionID)
			if err != nil {
				cfg.logger().ErrorContext(ctx, "Failed to load session settings", "error", err)
				settings = NewDefaultSettings(sessionID)
			}
			if settings == nil {
				settings = NewDefaultSettings(sessionID)
				if err := repo.Upsert(ctx, settings); err != nil {
					cfg.logger().ErrorContext(ctx, "Failed to create session settings", "error", err)
				}
			}

			if time.Since(settings.UpdatedAt) > 24*time.Hour {
				_ = repo.Touch(ctx, sessionID)
			}

			r = r.WithContext(context.WithValue(r.Context(), settingsCtxKey, settings))
			next.ServeHTTP(w, r)
		})
	}
}

// GetSettings returns the session settings from the request context.
func GetSettings(r *http.Request) *SessionSettings {
	if s, ok := r.Context().Value(settingsCtxKey).(*SessionSettings); ok {
		return s
	}
	return NewDefaultSettings("")
}

func getOrCreateSessionCookie(w http.ResponseWriter, r *http.Request, cookieName string) string {
	if cookie, err := r.Cookie(cookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	id := randomUUID()
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
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
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
