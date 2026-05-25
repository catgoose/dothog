// setup:feature:session_settings

package routes

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestHandleSessionsDelete_RemovesByUUIDAndRendersTable(t *testing.T) {
	store := &fakeSettingsStore{}
	ar := &AppRoutes{
		e:     echo.New(),
		deps: Deps{SessionStore: store, SessionSettings: store},
	}
	ar.e.DELETE("/admin/sessions/:uuid", ar.handleSessionsDelete)

	req := httptest.NewRequest(http.MethodDelete, "/admin/sessions/uuid-42", nil)
	rec := httptest.NewRecorder()
	ar.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "DELETE /admin/sessions/:uuid should succeed and return the refreshed table")
	require.Equal(t, []string{"uuid-42"}, store.deleted, "the admin contract's DeleteByUUID should receive the path uuid verbatim")
}

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
	store := &fakeSettingsStore{}
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
