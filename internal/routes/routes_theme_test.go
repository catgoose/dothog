// setup:feature:session_settings

package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"catgoose/dothog/internal/session"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// fakeSettingsStore captures the last upsert so tests can assert persistence
// without standing up a database. It satisfies both session.SettingsProvider
// and session.SettingsAdmin so a single fake can stand in for both Repos
// fields.
type fakeSettingsStore struct {
	last    *session.Settings
	deleted []string
	mu      sync.Mutex
	upserts int
}

func (s *fakeSettingsStore) GetByUUID(_ context.Context, _ string) (*session.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last == nil {
		return nil, nil
	}
	clone := *s.last
	return &clone, nil
}

func (s *fakeSettingsStore) Touch(_ context.Context, _ string) error {
	return nil
}

func (s *fakeSettingsStore) ListAll(_ context.Context) ([]session.Settings, error) {
	return nil, nil
}

func (s *fakeSettingsStore) DeleteByUUID(_ context.Context, uuid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted = append(s.deleted, uuid)
	return nil
}

func (s *fakeSettingsStore) Upsert(_ context.Context, settings *session.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	clone := *settings
	s.last = &clone
	s.upserts++
	return nil
}

func (s *fakeSettingsStore) lastSettings() *session.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last == nil {
		return nil
	}
	clone := *s.last
	return &clone
}

// withSession injects a Settings into the request context so handleTheme
// sees a session without standing up the full session middleware.
func withSession(req *http.Request, s *session.Settings) *http.Request {
	ctx := session.ContextWithSettings(req.Context(), s)
	return req.WithContext(ctx)
}

// TestInitThemeRoutes_WithoutBroker_PersistsTheme is the core regression:
// a scaffold without sse must still be able to register and serve
// POST /settings/theme so theme changes survive a reload.
func TestInitThemeRoutes_WithoutBroker_PersistsTheme(t *testing.T) {
	store := &fakeSettingsStore{}
	ar := &AppRoutes{
		e:     echo.New(),
		repos: Repos{SessionStore: store, SessionSettings: store},
	}
	ar.initThemeRoutes()

	form := url.Values{}
	form.Set("theme", "dark")
	req := httptest.NewRequest(http.MethodPost, "/settings/theme", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = withSession(req, session.NewDefaultSettings("test-uuid"))

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "POST /settings/theme should succeed without a broker")
	require.Equal(t, 1, store.upserts, "theme change should upsert once")
	last := store.lastSettings()
	require.NotNil(t, last)
	require.Equal(t, "dark", last.Theme, "upserted Settings should carry the new theme")
}

// TestInitThemeRoutes_WithoutBroker_RejectsInvalidTheme reuses the same shape
// to confirm the invalid-theme fallback path still works without a broker
// (and that persistence happens at the fallback value).
func TestInitThemeRoutes_WithoutBroker_RejectsInvalidTheme(t *testing.T) {
	store := &fakeSettingsStore{}
	ar := &AppRoutes{
		e:     echo.New(),
		repos: Repos{SessionStore: store, SessionSettings: store},
	}
	ar.initThemeRoutes()

	form := url.Values{}
	form.Set("theme", "no-such-theme")
	req := httptest.NewRequest(http.MethodPost, "/settings/theme", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = withSession(req, session.NewDefaultSettings("test-uuid"))

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	last := store.lastSettings()
	require.NotNil(t, last)
	require.Equal(t, "light", last.Theme, "invalid theme should fall back to 'light' and still persist")
}

// TestInitThemeRoutes_WithoutBroker_PersistsLayout guards the layout endpoint
// which lives on the same code path and was also previously broker-coupled.
func TestInitThemeRoutes_WithoutBroker_PersistsLayout(t *testing.T) {
	store := &fakeSettingsStore{}
	ar := &AppRoutes{
		e:     echo.New(),
		repos: Repos{SessionStore: store, SessionSettings: store},
	}
	ar.initThemeRoutes()

	form := url.Values{}
	form.Set("layout", session.LayoutApp)
	req := httptest.NewRequest(http.MethodPost, "/settings/layout", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = withSession(req, session.NewDefaultSettings("test-uuid"))

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "true", rec.Header().Get("HX-Refresh"), "layout change should ask the browser to refresh")
	last := store.lastSettings()
	require.NotNil(t, last)
	require.Equal(t, session.LayoutApp, last.Layout)
}
