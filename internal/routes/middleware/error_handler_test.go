package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"catgoose/dothog/internal/logger"
	"catgoose/dothog/internal/routes/hypermedia"

	"github.com/catgoose/promolog"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("LOG_LEVEL", "ERROR")
	logger.Init()
}

// setupEcho creates an Echo instance with promolog correlation and the HTTPErrorHandler.
func setupEcho(reqLogStore *promolog.Store) *echo.Echo {
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	e.HTTPErrorHandler = NewHTTPErrorHandler(reqLogStore)
	return e
}

// ---------------------------------------------------------------------------
// 1. No error from handler — response unchanged
// ---------------------------------------------------------------------------

func TestHTTPErrorHandler_NoError(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/ok", func(c echo.Context) error {
		return c.String(http.StatusOK, "all good")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "all good", rec.Body.String())
}

// ---------------------------------------------------------------------------
// 2. echo.HTTPError with HTMX request — returns HTML error component
// ---------------------------------------------------------------------------

func TestHTTPErrorHandler_EchoHTTPError_HTMX(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/test", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	body := rec.Body.String()
	require.True(t,
		strings.Contains(body, "not found") || strings.Contains(body, "error"),
		"expected HTML error content, got: %s", body,
	)
}

// ---------------------------------------------------------------------------
// 3. echo.HTTPError without HTMX request — returns HATEOAS HTML error page
// ---------------------------------------------------------------------------

func TestHTTPErrorHandler_EchoHTTPError_NonHTMX(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/test", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "bad request", "expected error message in HTML body")
	require.Contains(t, body, "<!doctype html>", "expected full HTML page for non-HTMX")
	require.Contains(t, body, "Close", "expected Dismiss control in error page")
}

// ---------------------------------------------------------------------------
// 4. hypermedia.HTTPError with HTMX — returns HTML with controls rendered
// ---------------------------------------------------------------------------

func TestHTTPErrorHandler_HypermediaHTTPError(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/test", func(c echo.Context) error {
		he := hypermedia.NewHTTPError(hypermedia.ErrorContext{
			StatusCode: 404,
			Message:    "not found",
			Route:      "/test",
			Closable:   true,
			Controls: []hypermedia.Control{
				hypermedia.BackButton("Go back"),
				hypermedia.GoHomeButton("Home", "/", "#main"),
			},
		})
		return he
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	body := rec.Body.String()
	require.True(t,
		strings.Contains(body, "not found") || strings.Contains(body, "error"),
		"expected HTML error content with controls, got: %s", body,
	)
}

// ---------------------------------------------------------------------------
// 5. hypermedia.HTTPError without HTMX — returns full HTML error page
// ---------------------------------------------------------------------------

func TestHTTPErrorHandler_HypermediaHTTPError_NonHTMX(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/test", func(c echo.Context) error {
		he := hypermedia.NewHTTPError(hypermedia.ErrorContext{
			StatusCode: 403,
			Message:    "forbidden",
			Route:      "/test",
			Closable:   true,
			Controls: []hypermedia.Control{
				hypermedia.BackButton("Go back"),
			},
		})
		return he
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "<!doctype html>", "expected full HTML page for non-HTMX")
	require.Contains(t, body, "forbidden", "expected error message in HTML body")
	require.Contains(t, body, "Go back", "expected handler-provided control")
	require.NotContains(t, body, "on click tell", "full page error should not have close button")
}

// ---------------------------------------------------------------------------
// 6. Generic error (non-HTTPError) — falls back to 500 with HTMX request
// ---------------------------------------------------------------------------

func TestHTTPErrorHandler_GenericError_HTMX(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/test", func(c echo.Context) error {
		return errors.New("internal failure")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "operation failed")
}

// ---------------------------------------------------------------------------
// 7. Generic error without HTMX — returns full HTML error page
// ---------------------------------------------------------------------------

func TestHTTPErrorHandler_GenericError_NonHTMX(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/test", func(c echo.Context) error {
		return errors.New("internal failure")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "<!doctype html>", "expected full HTML page for non-HTMX")
	require.Contains(t, body, "operation failed", "expected error message")
}

// ---------------------------------------------------------------------------
// 8. Response already committed — returns without modifying response
// ---------------------------------------------------------------------------

func TestHTTPErrorHandler_ResponseCommitted(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/test", func(c echo.Context) error {
		c.Response().WriteHeader(http.StatusOK)
		_, _ = c.Response().Write([]byte("already sent"))
		return echo.NewHTTPError(http.StatusInternalServerError, "ignored")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

// ---------------------------------------------------------------------------
// 9. Context canceled — returns without rendering
// ---------------------------------------------------------------------------

func TestHTTPErrorHandler_ContextCanceled(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = NewHTTPErrorHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("HX-Request", "true")

	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Call the error handler directly
	e.HTTPErrorHandler(echo.NewHTTPError(http.StatusNotFound, "not found"), c)

	require.Empty(t, rec.Body.String(),
		fmt.Sprintf("expected empty body when context canceled, got: %s", rec.Body.String()),
	)
}
