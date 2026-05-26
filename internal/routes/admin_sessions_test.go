// setup:feature:session_settings

package routes

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"catgoose/dothog/internal/session"
	// setup:feature:sse:start
	"time"

	"github.com/catgoose/tavern"
	// setup:feature:sse:end
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestHandleSessionsDelete_RemovesByUUIDAndRendersTable(t *testing.T) {
	store := &fakeSettingsStore{
		rows: map[string]*session.Settings{
			"uuid-42": {SessionUUID: "uuid-42", Theme: "dark"},
		},
	}
	ar := &AppRoutes{
		e:    echo.New(),
		deps: Deps{SessionStore: store, SessionSettings: store},
	}
	ar.e.DELETE("/admin/sessions/:uuid", ar.handleSessionsDelete)

	req := httptest.NewRequest(http.MethodDelete, "/admin/sessions/uuid-42", nil)
	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "DELETE /admin/sessions/:uuid should succeed and return the refreshed table")
	require.Equal(t, []string{"uuid-42"}, store.deleted, "the admin contract's DeleteByUUID should receive the path uuid verbatim")
}

// setup:feature:sse:start

func TestHandleSessionsDelete_PublishesDefaultThemeReset(t *testing.T) {
	store := &fakeSettingsStore{
		rows: map[string]*session.Settings{
			"uuid-42": {SessionUUID: "uuid-42", Theme: "dark"},
		},
	}
	broker := tavern.NewSSEBroker()
	defer broker.Close()

	ar := &AppRoutes{
		e:      echo.New(),
		deps:   Deps{SessionStore: store, SessionSettings: store},
		broker: broker,
	}
	ar.e.DELETE("/admin/sessions/:uuid", ar.handleSessionsDelete)

	msgs, unsub := broker.Subscribe(ThemeTopicForSession("uuid-42"))
	defer unsub()

	req := httptest.NewRequest(http.MethodDelete, "/admin/sessions/uuid-42", nil)
	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	select {
	case msg := <-msgs:
		require.Contains(t, msg, "theme-change")
		require.Contains(t, msg, session.DefaultTheme)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("deleted session should receive a default-theme reset event")
	}
}

// setup:feature:sse:end

func TestHandleSessionsDelete_MissingStore500s(t *testing.T) {
	ar := &AppRoutes{e: echo.New()}
	ar.e.DELETE("/admin/sessions/:uuid", ar.handleSessionsDelete)

	req := httptest.NewRequest(http.MethodDelete, "/admin/sessions/uuid-42", nil)
	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code, "no SessionSettings wired should surface a clear server error, not a panic")
}

// TestHandleSessionsBatchDelete_DeletesEachUUIDAndRendersTable pins the
// bulk-selection contract: each UUID in the comma-separated form field hits
// the admin store's DeleteByUUID and the response is the refreshed table.
func TestHandleSessionsBatchDelete_DeletesEachUUIDAndRendersTable(t *testing.T) {
	store := &fakeSettingsStore{
		rows: map[string]*session.Settings{
			"uuid-a": {SessionUUID: "uuid-a", Theme: "dark"},
			"uuid-b": {SessionUUID: "uuid-b", Theme: "night"},
			"uuid-c": {SessionUUID: "uuid-c", Theme: "acid"},
		},
	}
	ar := &AppRoutes{
		e:    echo.New(),
		deps: Deps{SessionStore: store, SessionSettings: store},
	}
	ar.e.POST("/admin/sessions/delete", ar.handleSessionsBatchDelete)

	form := url.Values{}
	form.Set("uuids", "uuid-a,uuid-b,uuid-c")
	req := httptest.NewRequest(http.MethodPost, "/admin/sessions/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "batch delete should succeed and render the refreshed table")
	require.Equal(t, []string{"uuid-a", "uuid-b", "uuid-c"}, store.deleted,
		"every uuid in the comma-separated payload should reach the admin store")
	require.Contains(t, rec.Body.String(), "Session UUID",
		"the response body should be the refreshed sessions table")
}

// setup:feature:sse:start

func TestHandleSessionsBatchDelete_PublishesDefaultThemeResetForEachDeletedRow(t *testing.T) {
	store := &fakeSettingsStore{
		rows: map[string]*session.Settings{
			"uuid-a": {SessionUUID: "uuid-a", Theme: "dark"},
			"uuid-b": {SessionUUID: "uuid-b", Theme: "night"},
		},
	}
	broker := tavern.NewSSEBroker()
	defer broker.Close()

	ar := &AppRoutes{
		e:      echo.New(),
		deps:   Deps{SessionStore: store, SessionSettings: store},
		broker: broker,
	}
	ar.e.POST("/admin/sessions/delete", ar.handleSessionsBatchDelete)

	aMsgs, aUnsub := broker.Subscribe(ThemeTopicForSession("uuid-a"))
	defer aUnsub()
	bMsgs, bUnsub := broker.Subscribe(ThemeTopicForSession("uuid-b"))
	defer bUnsub()

	form := url.Values{}
	form.Set("uuids", "uuid-a,uuid-b")
	req := httptest.NewRequest(http.MethodPost, "/admin/sessions/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	for name, msgs := range map[string]<-chan string{"uuid-a": aMsgs, "uuid-b": bMsgs} {
		select {
		case msg := <-msgs:
			require.Contains(t, msg, "theme-change", "%s should receive a theme reset event", name)
			require.Contains(t, msg, session.DefaultTheme, "%s should be reset to default theme", name)
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("%s should receive a default-theme reset event", name)
		}
	}
}

// setup:feature:sse:end

// TestHandleSessionsBatchDelete_EmptyPayloadJustRendersTable keeps the
// no-rows-selected path honest: a missing or blank uuids field is not an
// error, the store sees no deletes, and the response is still the table.
func TestHandleSessionsBatchDelete_EmptyPayloadJustRendersTable(t *testing.T) {
	store := &fakeSettingsStore{}
	ar := &AppRoutes{
		e:    echo.New(),
		deps: Deps{SessionStore: store, SessionSettings: store},
	}
	ar.e.POST("/admin/sessions/delete", ar.handleSessionsBatchDelete)

	form := url.Values{}
	form.Set("uuids", "")
	req := httptest.NewRequest(http.MethodPost, "/admin/sessions/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, store.deleted, "empty uuids field must not reach DeleteByUUID")
}

// TestHandleSessionsBatchDelete_TrimsAndSkipsBlanks defends the parser
// against stray whitespace and double-commas a client might emit when
// stringifying its selection set.
func TestHandleSessionsBatchDelete_TrimsAndSkipsBlanks(t *testing.T) {
	store := &fakeSettingsStore{}
	ar := &AppRoutes{
		e:    echo.New(),
		deps: Deps{SessionStore: store, SessionSettings: store},
	}
	ar.e.POST("/admin/sessions/delete", ar.handleSessionsBatchDelete)

	form := url.Values{}
	form.Set("uuids", " uuid-a , , uuid-b ")
	req := httptest.NewRequest(http.MethodPost, "/admin/sessions/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []string{"uuid-a", "uuid-b"}, store.deleted)
}

// TestHandleSessionsBatchDelete_MissingStore500s mirrors the single-delete
// guardrail: bulk delete without a wired admin store fails with a clear 500.
func TestHandleSessionsBatchDelete_MissingStore500s(t *testing.T) {
	ar := &AppRoutes{e: echo.New()}
	ar.e.POST("/admin/sessions/delete", ar.handleSessionsBatchDelete)

	form := url.Values{}
	form.Set("uuids", "uuid-a")
	req := httptest.NewRequest(http.MethodPost, "/admin/sessions/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleSessionsUpdateTheme_PersistsAndRendersTable(t *testing.T) {
	store := &fakeSettingsStore{
		rows: map[string]*session.Settings{
			"uuid-42": {SessionUUID: "uuid-42", Theme: "light"},
		},
	}
	ar := &AppRoutes{
		e:    echo.New(),
		deps: Deps{SessionStore: store, SessionSettings: store},
	}
	ar.e.POST("/admin/sessions/:uuid/theme", ar.handleSessionsUpdateTheme)

	form := url.Values{}
	form.Set("theme", "dark")
	req := httptest.NewRequest(http.MethodPost, "/admin/sessions/uuid-42/theme", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []themeUpdateCall{{uuid: "uuid-42", theme: "dark"}}, store.themeUpdateCalls())
	require.Contains(t, rec.Body.String(), "Session UUID")
}

// setup:feature:sse:start

func TestHandleSessionsUpdateTheme_PublishesToSessionTopic(t *testing.T) {
	store := &fakeSettingsStore{
		rows: map[string]*session.Settings{
			"uuid-42": {SessionUUID: "uuid-42", Theme: "light"},
		},
	}
	broker := tavern.NewSSEBroker()
	defer broker.Close()

	ar := &AppRoutes{
		e:      echo.New(),
		deps:   Deps{SessionStore: store, SessionSettings: store},
		broker: broker,
	}
	ar.e.POST("/admin/sessions/:uuid/theme", ar.handleSessionsUpdateTheme)

	targetMsgs, targetUnsub := broker.Subscribe(ThemeTopicForSession("uuid-42"))
	defer targetUnsub()
	bystanderMsgs, bystanderUnsub := broker.Subscribe(ThemeTopicForSession("uuid-b"))
	defer bystanderUnsub()

	form := url.Values{}
	form.Set("theme", "dark")
	req := httptest.NewRequest(http.MethodPost, "/admin/sessions/uuid-42/theme", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	select {
	case msg := <-targetMsgs:
		require.Contains(t, msg, "theme-change")
		require.Contains(t, msg, "dark")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("target session topic did not receive the theme-change event")
	}

	select {
	case msg := <-bystanderMsgs:
		t.Fatalf("bystander session topic must not receive the theme-change event; got %q", msg)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestHandleSessionsUpdateTheme_StaleRowSuppressesPublish(t *testing.T) {
	store := &fakeSettingsStore{
		missingUUIDs: map[string]bool{"vanished": true},
	}
	broker := tavern.NewSSEBroker()
	defer broker.Close()

	ar := &AppRoutes{
		e:      echo.New(),
		deps:   Deps{SessionStore: store, SessionSettings: store},
		broker: broker,
	}
	ar.e.POST("/admin/sessions/:uuid/theme", ar.handleSessionsUpdateTheme)

	msgs, unsub := broker.Subscribe(ThemeTopicForSession("vanished"))
	defer unsub()

	form := url.Values{}
	form.Set("theme", "dark")
	req := httptest.NewRequest(http.MethodPost, "/admin/sessions/vanished/theme", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []themeUpdateCall{{uuid: "vanished", theme: "dark"}}, store.themeUpdateCalls())

	select {
	case msg := <-msgs:
		t.Fatalf("stale row save must not publish a theme-change event; got %q", msg)
	case <-time.After(100 * time.Millisecond):
	}
}

// setup:feature:sse:end

func TestHandleSessionsUpdateTheme_InvalidTheme400s(t *testing.T) {
	store := &fakeSettingsStore{}
	ar := &AppRoutes{
		e:    echo.New(),
		deps: Deps{SessionStore: store, SessionSettings: store},
	}
	ar.e.POST("/admin/sessions/:uuid/theme", ar.handleSessionsUpdateTheme)

	form := url.Values{}
	form.Set("theme", "bogus")
	req := httptest.NewRequest(http.MethodPost, "/admin/sessions/uuid-42/theme", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleSessionsUpdateTheme_MissingStore500s(t *testing.T) {
	ar := &AppRoutes{e: echo.New()}
	ar.e.POST("/admin/sessions/:uuid/theme", ar.handleSessionsUpdateTheme)

	form := url.Values{}
	form.Set("theme", "dark")
	req := httptest.NewRequest(http.MethodPost, "/admin/sessions/uuid-42/theme", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
