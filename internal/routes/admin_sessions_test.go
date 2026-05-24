// setup:feature:session_settings

package routes

import (
	"net/http"
	"net/http/httptest"
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
