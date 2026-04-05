// setup:feature:session_settings

package session

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	appenv "catgoose/dothog/internal/env"
	"time"
)

type settingsKeyType struct{}

var settingsCtxKey settingsKeyType

// Settings holds per-session user preferences keyed by a browser UUID
// cookie. The struct is designed to be stored in a database row; the db tags
// match the expected column names.
type Settings struct {
	UpdatedAt   time.Time         `db:"UpdatedAt"`
	Extra       map[string]string `db:"Extra" json:"extra,omitempty"`
	SessionUUID string            `db:"SessionUUID"`
	Theme       string            `db:"Theme"`
	Layout      string            `db:"Layout"`
	ID          int               `db:"Id"`
}

// GetExtra returns the value for key, or empty string if not set.
func (s *Settings) GetExtra(key string) string {
	if s.Extra == nil {
		return ""
	}
	return s.Extra[key]
}

// SetExtra sets a key-value pair in Extra, initializing the map if nil.
func (s *Settings) SetExtra(key, value string) {
	if s.Extra == nil {
		s.Extra = make(map[string]string)
	}
	s.Extra[key] = value
}

// MarshalExtra returns the Extra map serialized as a JSON string.
func (s *Settings) MarshalExtra() (string, error) {
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
func (s *Settings) UnmarshalExtra(data string) error {
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

// Session settings defaults.
const (
	DefaultTheme  = "light"
	DefaultLayout = "classic"
	LayoutApp     = "app"
)

// NewDefaultSettings returns a Settings with defaults for the given UUID.
func NewDefaultSettings(uuid string) *Settings {
	return &Settings{
		SessionUUID: uuid,
		Theme:       DefaultTheme,
		Layout:      DefaultLayout,
		Extra:       make(map[string]string),
	}
}

// Config holds session middleware configuration.
type Config struct {
	Logger     *slog.Logger
	CookieName string
}

func (cfg Config) cookieName() string {
	if cfg.CookieName != "" {
		return cfg.CookieName
	}
	return "session_id"
}

func (cfg Config) logger() *slog.Logger {
	if cfg.Logger != nil {
		return cfg.Logger
	}
	return slog.Default()
}

// Provider is the interface for session-settings persistence.
type Provider interface {
	GetByUUID(ctx context.Context, uuid string) (*Settings, error)
	Upsert(ctx context.Context, s *Settings) error
	Touch(ctx context.Context, uuid string) error
}

// IDFunc returns the session identifier for the current request.
type IDFunc func(r *http.Request) string

// Middleware returns middleware that loads per-session settings and stores
// them on the request context for downstream handlers.
func Middleware(repo Provider, idFunc IDFunc, cfgs ...Config) func(http.Handler) http.Handler {
	var cfg Config
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
				var err error
				sessionID, err = getOrCreateSessionCookie(w, r, cfg.cookieName())
				if err != nil {
					cfg.logger().ErrorContext(ctx, "Failed to create session cookie", "error", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
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
func GetSettings(r *http.Request) *Settings {
	if s, ok := r.Context().Value(settingsCtxKey).(*Settings); ok {
		return s
	}
	return NewDefaultSettings("")
}

func getOrCreateSessionCookie(w http.ResponseWriter, r *http.Request, cookieName string) (string, error) {
	if cookie, err := r.Cookie(cookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}
	id, err := randomUUID()
	if err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    id,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   !appenv.Dev(),
		SameSite: http.SameSiteLaxMode,
	})
	return id, nil
}

func randomUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session ID: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
