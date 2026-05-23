package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubProvider struct{}

func (stubProvider) GetByUUID(context.Context, string) (*Settings, error) { return nil, nil }
func (stubProvider) Upsert(context.Context, *Settings) error              { return nil }
func (stubProvider) Touch(context.Context, string) error                  { return nil }

type stubAdmin struct{}

func (stubAdmin) ListAll(context.Context) ([]Settings, error)          { return nil, nil }
func (stubAdmin) GetByUUID(context.Context, string) (*Settings, error) { return nil, nil }
func (stubAdmin) DeleteByUUID(context.Context, string) error           { return nil }

// Compile-time proof that the contracts are split: middleware never sees
// DeleteByUUID, admin never sees Touch.
var (
	_ SettingsProvider = stubProvider{}
	_ SettingsAdmin    = stubAdmin{}
)

func TestConfigCookieNameSanitizesAppLabel(t *testing.T) {
	cfg := Config{CookieName: "My App_session_id"}
	require.Equal(t, "my_app_session_id", cfg.cookieName())
}

func TestDefaultCookieNameFromAppName(t *testing.T) {
	require.Equal(t, "my_app_session_id", DefaultCookieName("My App"))
	require.Equal(t, "sales_app_2_session_id", DefaultCookieName("Sales-App 2"))
}

func TestConfigCookieNamePreservesValidCookieTokens(t *testing.T) {
	cfg := Config{CookieName: "__Host-App.Session"}
	require.Equal(t, "__host-app.session", cfg.cookieName())
}

func TestIsValidCookieName(t *testing.T) {
	require.True(t, IsValidCookieName("__Host-App.Session"))
	require.False(t, IsValidCookieName("My App_session_id"))
	require.False(t, IsValidCookieName(""))
}

func TestMiddlewareSetsSanitizedCookieName(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	mw := Middleware(stubProvider{}, nil, Config{CookieName: "My App_session_id"})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	cookies := res.Cookies()
	require.NotEmpty(t, cookies)
	require.Equal(t, "my_app_session_id", cookies[0].Name)
	require.NotEmpty(t, cookies[0].Value)
}
