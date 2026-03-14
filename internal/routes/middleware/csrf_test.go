// setup:feature:csrf
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/catgoose/crooner"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

type mockSessionManager struct {
	data map[string]any
}

func (m *mockSessionManager) Get(c echo.Context, key string) (any, error) {
	if m.data == nil {
		return nil, nil
	}
	return m.data[key], nil
}

func (m *mockSessionManager) Set(c echo.Context, key string, value any) error {
	if m.data == nil {
		m.data = make(map[string]any)
	}
	m.data[key] = value
	return nil
}

func (m *mockSessionManager) Delete(c echo.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockSessionManager) Clear(c echo.Context) error {
	m.data = nil
	return nil
}

func (m *mockSessionManager) Invalidate(c echo.Context) error {
	return nil
}

func (m *mockSessionManager) ClearInvalidate(c echo.Context) error {
	m.data = nil
	return nil
}

func TestCSRF_NilSessionManager_NoOp(t *testing.T) {
	e := echo.New()
	e.Use(CSRF(nil, CSRFConfig{}))
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Body.String())
}

func TestCSRF_GET_CreatesToken(t *testing.T) {
	sm := &mockSessionManager{data: make(map[string]any)}
	e := echo.New()
	e.Use(CSRF(sm, CSRFConfig{}))
	e.GET("/", func(c echo.Context) error {
		token, ok := c.Get(csrfContextKey).(string)
		require.True(t, ok)
		require.NotEmpty(t, token)
		return c.String(http.StatusOK, token)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	token := rec.Body.String()
	require.Len(t, token, 64)
	sessionToken, _ := sm.Get(nil, csrfTokenSessionKey)
	require.Equal(t, token, sessionToken)
}

func TestCSRF_POST_WithoutToken_403(t *testing.T) {
	sm := &mockSessionManager{data: make(map[string]any)}
	e := echo.New()
	e.Use(CSRF(sm, CSRFConfig{}))
	e.POST("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCSRF_POST_WithValidToken_200(t *testing.T) {
	sm := &mockSessionManager{data: make(map[string]any)}
	require.NoError(t, sm.Set(nil, csrfTokenSessionKey, "abc123token"))
	e := echo.New()
	e.Use(CSRF(sm, CSRFConfig{}))
	e.POST("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-CSRF-Token", "abc123token")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestCSRF_POST_WithWrongToken_403(t *testing.T) {
	sm := &mockSessionManager{data: make(map[string]any)}
	require.NoError(t, sm.Set(nil, csrfTokenSessionKey, "righttoken"))
	e := echo.New()
	e.Use(CSRF(sm, CSRFConfig{}))
	e.POST("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-CSRF-Token", "wrongtoken")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCSRF_ExemptPath_SkipsValidation(t *testing.T) {
	sm := &mockSessionManager{data: make(map[string]any)}
	e := echo.New()
	e.Use(CSRF(sm, CSRFConfig{ExemptPaths: []string{"/login", "/callback", "/logout"}}))
	e.POST("/login", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestCSRF_PerRequestPath_RotatesToken(t *testing.T) {
	sm := &mockSessionManager{data: make(map[string]any)}
	e := echo.New()
	e.Use(CSRF(sm, CSRFConfig{PerRequestPaths: []string{"/form"}}))
	e.GET("/form", func(c echo.Context) error {
		token, _ := c.Get(csrfContextKey).(string)
		return c.String(http.StatusOK, token)
	})

	req1 := httptest.NewRequest(http.MethodGet, "/form", nil)
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)
	token1 := rec1.Body.String()

	req2 := httptest.NewRequest(http.MethodGet, "/form", nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusOK, rec2.Code)
	token2 := rec2.Body.String()

	require.NotEqual(t, token1, token2)
}

func Test_pathExempt(t *testing.T) {
	exempt := []string{"/login", "/callback", "/logout"}
	require.True(t, pathExempt(exempt, "/login"))
	require.True(t, pathExempt(exempt, "/callback"))
	require.True(t, pathExempt(exempt, "/logout"))
	require.False(t, pathExempt(exempt, "/"))
	require.False(t, pathExempt(exempt, "/api/login"))
}

func Test_pathPerRequest(t *testing.T) {
	paths := []string{"/form", "/edit"}
	require.True(t, pathPerRequest(paths, "/form"))
	require.True(t, pathPerRequest(paths, "/edit"))
	require.False(t, pathPerRequest(paths, "/"))
	require.False(t, pathPerRequest(paths, "/form/1"))
}

var _ crooner.SessionManager = (*mockSessionManager)(nil)
