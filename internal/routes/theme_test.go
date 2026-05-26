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
	// setup:feature:sse:start
	"time"

	"github.com/catgoose/tavern"
	// setup:feature:sse:end
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// fakeSettingsStore captures the last upsert so tests can assert persistence
// without standing up a database. It satisfies both session.SettingsProvider
// and session.SettingsAdmin so a single fake can stand in for both Deps
// fields.
type fakeSettingsStore struct {
	last         *session.Settings
	rows         map[string]*session.Settings
	deleted      []string
	themeUpdates []themeUpdateCall
	missingUUIDs map[string]bool
	mu           sync.Mutex
	upserts      int
}

type themeUpdateCall struct {
	uuid  string
	theme string
}

func (s *fakeSettingsStore) GetByUUID(_ context.Context, uuid string) (*session.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rows != nil {
		forUUID := s.rows[uuid]
		if forUUID == nil {
			return nil, nil
		}
		clone := *forUUID
		return &clone, nil
	}
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
	if s.rows != nil {
		delete(s.rows, uuid)
	}
	return nil
}

func (s *fakeSettingsStore) UpdateThemeByUUID(_ context.Context, uuid, theme string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.themeUpdates = append(s.themeUpdates, themeUpdateCall{uuid: uuid, theme: theme})
	if s.missingUUIDs != nil && s.missingUUIDs[uuid] {
		return false, nil
	}
	if s.rows != nil {
		row := s.rows[uuid]
		if row == nil {
			return false, nil
		}
		clone := *row
		clone.Theme = theme
		s.rows[uuid] = &clone
	}
	return true, nil
}

func (s *fakeSettingsStore) Upsert(_ context.Context, settings *session.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	clone := *settings
	s.last = &clone
	s.upserts++
	if s.rows != nil {
		s.rows[settings.SessionUUID] = &clone
	}
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

func (s *fakeSettingsStore) themeUpdateCalls() []themeUpdateCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]themeUpdateCall, len(s.themeUpdates))
	copy(out, s.themeUpdates)
	return out
}

// withSession injects a Settings into the request context so handleTheme
// sees a session without standing up the full session middleware.
func withSession(req *http.Request, s *session.Settings) *http.Request {
	ctx := session.ContextWithSettings(req.Context(), s)
	return req.WithContext(ctx)
}

// TestInitThemeRoutes_WithoutBroker_PersistsTheme keeps the non-HTMX fallback
// honest: POST /settings/theme persists then redirects back to /settings.
func TestInitThemeRoutes_WithoutBroker_PersistsTheme(t *testing.T) {
	store := &fakeSettingsStore{}
	ar := &AppRoutes{
		e:    echo.New(),
		deps: Deps{SessionStore: store, SessionSettings: store},
	}
	ar.initThemeRoutes()

	form := url.Values{}
	form.Set("theme", "dark")
	req := httptest.NewRequest(http.MethodPost, "/settings/theme", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = withSession(req, session.NewDefaultSettings("test-uuid"))

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusSeeOther, rec.Code, "plain POST /settings/theme should redirect back to settings")
	require.Equal(t, "/settings", rec.Header().Get("Location"))
	require.Equal(t, 1, store.upserts, "theme change should upsert once")
	last := store.lastSettings()
	require.NotNil(t, last)
	require.Equal(t, "dark", last.Theme, "upserted Settings should carry the new theme")
}

// TestInitThemeRoutes_WithoutBroker_HTMXThemeChangeReturnsPicker keeps the
// server-driven seam honest: htmx requests get the canonical picker fragment
// back plus the theme-change trigger header.
func TestInitThemeRoutes_WithoutBroker_HTMXThemeChangeReturnsPicker(t *testing.T) {
	store := &fakeSettingsStore{}
	ar := &AppRoutes{
		e:    echo.New(),
		deps: Deps{SessionStore: store, SessionSettings: store},
	}
	ar.initThemeRoutes()

	form := url.Values{}
	form.Set("theme", "dark")
	req := httptest.NewRequest(http.MethodPost, "/settings/theme", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req = withSession(req, session.NewDefaultSettings("test-uuid"))

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("HX-Trigger"), "app:theme-change")
	require.Contains(t, rec.Body.String(), "data-theme-picker")
	require.Contains(t, rec.Body.String(), `<option value="dark" selected>dark</option>`)
	last := store.lastSettings()
	require.NotNil(t, last)
	require.Equal(t, "dark", last.Theme)
}

// setup:feature:sse:start

// TestInitThemeRoutes_WithBroker_HTMXThemeChangeUsesSendOnlyResponse keeps
// the SSE-enabled flow honest: the POST is only the send channel and the UI
// update comes back over SSE, so the response itself is an empty 204 no-swap ack.
func TestInitThemeRoutes_WithBroker_HTMXThemeChangeUsesSendOnlyResponse(t *testing.T) {
	store := &fakeSettingsStore{}
	broker := tavern.NewSSEBroker()
	defer broker.Close()

	ar := &AppRoutes{
		e:      echo.New(),
		deps:   Deps{SessionStore: store, SessionSettings: store},
		broker: broker,
	}
	ar.initThemeRoutes()

	form := url.Values{}
	form.Set("theme", "dark")
	req := httptest.NewRequest(http.MethodPost, "/settings/theme", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req = withSession(req, session.NewDefaultSettings("test-uuid"))

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "none", rec.Header().Get("HX-Reswap"))
	require.Empty(t, rec.Header().Get("HX-Trigger"))
	require.Empty(t, rec.Body.String())
	last := store.lastSettings()
	require.NotNil(t, last)
	require.Equal(t, "dark", last.Theme)
}

// TestInitThemeRoutes_PublishScopedToSessionTopic verifies the per-session SSE
// contract: a POST /settings/theme for session A publishes only to A's topic,
// so a subscriber on session B's topic sees nothing.
func TestInitThemeRoutes_PublishScopedToSessionTopic(t *testing.T) {
	store := &fakeSettingsStore{}
	broker := tavern.NewSSEBroker()
	defer broker.Close()

	ar := &AppRoutes{
		e:      echo.New(),
		deps:   Deps{SessionStore: store, SessionSettings: store},
		broker: broker,
	}
	ar.initThemeRoutes()

	aMsgs, aUnsub := broker.Subscribe(ThemeTopicForSession("session-a"))
	defer aUnsub()
	bMsgs, bUnsub := broker.Subscribe(ThemeTopicForSession("session-b"))
	defer bUnsub()

	form := url.Values{}
	form.Set("theme", "dark")
	req := httptest.NewRequest(http.MethodPost, "/settings/theme", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req = withSession(req, session.NewDefaultSettings("session-a"))

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)

	select {
	case msg := <-aMsgs:
		require.Contains(t, msg, "theme-change", "session-a's topic should receive its own theme-change event")
		require.Contains(t, msg, "dark", "the published payload should carry the chosen theme")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("session-a's topic did not receive the theme-change event")
	}

	select {
	case msg := <-bMsgs:
		t.Fatalf("session-b's topic must not receive session-a's theme change; got %q", msg)
	case <-time.After(100 * time.Millisecond):
	}
}

// TestInitThemeRoutes_NoPublishWhenSessionUUIDMissing keeps the defensive
// gate honest: with a nil-UUID session (no cookie yet), the handler still
// persists/redirects but does not publish to any topic.
func TestInitThemeRoutes_NoPublishWhenSessionUUIDMissing(t *testing.T) {
	store := &fakeSettingsStore{}
	broker := tavern.NewSSEBroker()
	defer broker.Close()

	ar := &AppRoutes{
		e:      echo.New(),
		deps:   Deps{SessionStore: store, SessionSettings: store},
		broker: broker,
	}
	ar.initThemeRoutes()

	form := url.Values{}
	form.Set("theme", "dark")
	req := httptest.NewRequest(http.MethodPost, "/settings/theme", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = withSession(req, session.NewDefaultSettings(""))

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusSeeOther, rec.Code, "no SSE response shape change when UUID is missing")
	require.Equal(t, 0, broker.SubscriberCount(),
		"no session UUID means no subscribers and no publishes")
}

// setup:feature:sse:end

// TestInitThemeRoutes_WithoutBroker_RejectsInvalidTheme confirms the
// invalid-theme fallback still persists and re-renders at the canonical
// fallback value.
func TestInitThemeRoutes_WithoutBroker_RejectsInvalidTheme(t *testing.T) {
	store := &fakeSettingsStore{}
	ar := &AppRoutes{
		e:    echo.New(),
		deps: Deps{SessionStore: store, SessionSettings: store},
	}
	ar.initThemeRoutes()

	form := url.Values{}
	form.Set("theme", "no-such-theme")
	req := httptest.NewRequest(http.MethodPost, "/settings/theme", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req = withSession(req, session.NewDefaultSettings("test-uuid"))

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	last := store.lastSettings()
	require.NotNil(t, last)
	require.Equal(t, "light", last.Theme, "invalid theme should fall back to 'light' and still persist")
	require.Contains(t, rec.Body.String(), `<option value="light" selected>light</option>`)
}

// TestInitThemeRoutes_WithoutBroker_ThemePickerFragmentUsesSessionTheme pins
// the GET /settings/theme/picker route's contract: it renders the canonical
// picker fragment for the current session theme. The fragment is a plain
// data-theme-picker form (no Alpine seam) — theme-controller.js takes over
// via delegated submit/change/click events on [data-theme-picker].
func TestInitThemeRoutes_WithoutBroker_ThemePickerFragmentUsesSessionTheme(t *testing.T) {
	store := &fakeSettingsStore{}
	ar := &AppRoutes{
		e:    echo.New(),
		deps: Deps{SessionStore: store, SessionSettings: store},
	}
	ar.initThemeRoutes()

	req := httptest.NewRequest(http.MethodGet, "/settings/theme/picker", nil)
	req.Header.Set("HX-Request", "true")
	settings := session.NewDefaultSettings("test-uuid")
	settings.Theme = "forest"
	req = withSession(req, settings)

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, `data-theme-picker`,
		"picker fragment must keep the data-theme-picker marker theme-controller.js delegates from")
	require.Contains(t, body, `<option value="forest" selected>forest</option>`)
	require.NotContains(t, body, "app:theme-picker-sync",
		"picker should not re-fetch via the dropped app:theme-picker-sync event after the controller's syncPicker already mutates the DOM in place")
}

// TestInitThemeRoutes_WithoutBroker_PersistsLayout guards that the layout
// endpoint shares the theme endpoint's code path and persists with a nil SSE
// broker.
func TestInitThemeRoutes_WithoutBroker_PersistsLayout(t *testing.T) {
	store := &fakeSettingsStore{}
	ar := &AppRoutes{
		e:    echo.New(),
		deps: Deps{SessionStore: store, SessionSettings: store},
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
