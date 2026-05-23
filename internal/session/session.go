// setup:feature:session_settings

package session

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

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

// GetExtra is "" when key is missing or Extra is nil.
func (s *Settings) GetExtra(key string) string {
	if s.Extra == nil {
		return ""
	}
	return s.Extra[key]
}

// SetExtra lazily allocates Extra on first call.
func (s *Settings) SetExtra(key, value string) {
	if s.Extra == nil {
		s.Extra = make(map[string]string)
	}
	s.Extra[key] = value
}

// MarshalExtra serialises Extra to a JSON object; emits "{}" when Extra is nil.
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

// NewDefaultSettings seeds Theme=DefaultTheme, Layout=DefaultLayout, and an empty Extra map.
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
	name := cfg.CookieName
	if name == "" {
		name = "session_id"
	}
	return sanitizeCookieName(name)
}

// DefaultCookieName derives a stable snake_case session cookie name from appName.
func DefaultCookieName(appName string) string {
	appName = strings.TrimSpace(strings.ToLower(appName))
	if appName == "" {
		return "session_id"
	}

	var b strings.Builder
	lastWasUnderscore := false
	for _, r := range appName {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastWasUnderscore = false
		default:
			if b.Len() == 0 || lastWasUnderscore {
				continue
			}
			b.WriteByte('_')
			lastWasUnderscore = true
		}
	}

	base := strings.Trim(b.String(), "_")
	if base == "" {
		return "session_id"
	}
	return base + "_session_id"
}

// IsValidCookieName reports whether name contains only RFC 6265 token characters.
func IsValidCookieName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !isCookieTokenRune(r) {
			return false
		}
	}
	return true
}

func (cfg Config) logger() *slog.Logger {
	if cfg.Logger != nil {
		return cfg.Logger
	}
	return slog.Default()
}

// SettingsProvider is the middleware-facing contract: load a row by UUID,
// upsert one back, and bump UpdatedAt without rewriting the rest. It is the
// minimum surface session.Middleware needs to hydrate the request context.
type SettingsProvider interface {
	GetByUUID(ctx context.Context, uuid string) (*Settings, error)
	Upsert(ctx context.Context, s *Settings) error
	Touch(ctx context.Context, uuid string) error
}

// SettingsAdmin is the route-facing contract for /admin/sessions-style
// management surfaces: enumerate rows, fetch one, drop one. Implementations
// usually also satisfy SettingsProvider, but admin routes depend on this
// narrower interface so the extension seam stays obvious.
type SettingsAdmin interface {
	ListAll(ctx context.Context) ([]Settings, error)
	GetByUUID(ctx context.Context, uuid string) (*Settings, error)
	DeleteByUUID(ctx context.Context, uuid string) error
}

// IDFunc resolves the session identifier from r; "" triggers cookie-based ID derivation.
type IDFunc func(r *http.Request) string

// Middleware loads per-session settings via repo and stashes them on the request context
// for GetSettings; refreshes UpdatedAt via Touch once a day.
func Middleware(repo SettingsProvider, idFunc IDFunc, cfgs ...Config) func(http.Handler) http.Handler {
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

// GetSettings reads the context value populated by Middleware; falls back to defaults with "" UUID.
func GetSettings(r *http.Request) *Settings {
	if s, ok := r.Context().Value(settingsCtxKey).(*Settings); ok {
		return s
	}
	return NewDefaultSettings("")
}

// ContextWithSettings seeds the GetSettings key on ctx; useful in tests that bypass Middleware.
func ContextWithSettings(ctx context.Context, s *Settings) context.Context {
	return context.WithValue(ctx, settingsCtxKey, s)
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

func sanitizeCookieName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "session_id"
	}

	var b strings.Builder
	pendingSeparator := false
	for _, r := range name {
		if isCookieTokenRune(r) {
			if pendingSeparator && b.Len() > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
			pendingSeparator = false
			continue
		}
		if b.Len() > 0 {
			pendingSeparator = true
		}
	}

	sanitized := b.String()
	if sanitized == "" {
		return "session_id"
	}
	return sanitized
}

func isCookieTokenRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	}
	return strings.ContainsRune("!#$%&'*+-.^_`|~", r)
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
