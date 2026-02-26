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

	"catgoose/go-htmx-template/internals/logger"
	"catgoose/go-htmx-template/internals/routes/hypermedia"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("LOG_LEVEL", "ERROR")
	logger.Init()
}

// ---------------------------------------------------------------------------
// 1. No error from handler — middleware returns nil, response unchanged
// ---------------------------------------------------------------------------

func TestErrorHandlerMiddleware_NoError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := ErrorHandlerMiddleware()
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "all good")
	})

	err := handler(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "all good", rec.Body.String())
}

// ---------------------------------------------------------------------------
// 2. echo.HTTPError with HTMX request — returns HTML error component
// ---------------------------------------------------------------------------

func TestErrorHandlerMiddleware_EchoHTTPError_HTMX(t *testing.T) {
	e := echo.New()
	e.Use(RequestIDMiddleware())
	e.Use(ErrorHandlerMiddleware())
	e.GET("/test", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	// The response body should contain HTML from the error component
	body := rec.Body.String()
	require.True(t,
		strings.Contains(body, "not found") || strings.Contains(body, "error"),
		"expected HTML error content, got: %s", body,
	)
}

// ---------------------------------------------------------------------------
// 3. echo.HTTPError without HTMX request — returns HATEOAS HTML error page
// ---------------------------------------------------------------------------

func TestErrorHandlerMiddleware_EchoHTTPError_NonHTMX(t *testing.T) {
	e := echo.New()
	e.Use(RequestIDMiddleware())
	e.Use(ErrorHandlerMiddleware())
	e.GET("/test", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	body := rec.Body.String()
	// Should render an HTML error page with HATEOAS controls, not JSON
	require.Contains(t, body, "bad request", "expected error message in HTML body")
	require.Contains(t, body, "<!doctype html>", "expected full HTML page for non-HTMX")
	require.Contains(t, body, "Go Back", "expected Back control in error page")
	require.Contains(t, body, "Go Home", "expected Home control in error page")
}

// ---------------------------------------------------------------------------
// 4. hypermedia.HTTPError with HTMX — returns HTML with controls rendered
// ---------------------------------------------------------------------------

func TestErrorHandlerMiddleware_HypermediaHTTPError(t *testing.T) {
	e := echo.New()
	e.Use(RequestIDMiddleware())
	e.Use(ErrorHandlerMiddleware())
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

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
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

func TestErrorHandlerMiddleware_HypermediaHTTPError_NonHTMX(t *testing.T) {
	e := echo.New()
	e.Use(RequestIDMiddleware())
	e.Use(ErrorHandlerMiddleware())
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

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "<!doctype html>", "expected full HTML page for non-HTMX")
	require.Contains(t, body, "forbidden", "expected error message in HTML body")
	require.Contains(t, body, "Go back", "expected handler-provided control")
	// Closable should be forced to false for full page — no close button rendered
	require.NotContains(t, body, "on click tell", "full page error should not have close button")
}

// ---------------------------------------------------------------------------
// 6. Generic error (non-HTTPError) — falls back to 500 with HTMX request
// ---------------------------------------------------------------------------

func TestErrorHandlerMiddleware_GenericError_HTMX(t *testing.T) {
	e := echo.New()
	e.Use(RequestIDMiddleware())
	e.Use(ErrorHandlerMiddleware())
	e.GET("/test", func(c echo.Context) error {
		return errors.New("internal failure")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "operation failed")
}

// ---------------------------------------------------------------------------
// 7. Generic error without HTMX — returns full HTML error page
// ---------------------------------------------------------------------------

func TestErrorHandlerMiddleware_GenericError_NonHTMX(t *testing.T) {
	e := echo.New()
	e.Use(RequestIDMiddleware())
	e.Use(ErrorHandlerMiddleware())
	e.GET("/test", func(c echo.Context) error {
		return errors.New("internal failure")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "<!doctype html>", "expected full HTML page for non-HTMX")
	require.Contains(t, body, "operation failed", "expected error message")
}

// ---------------------------------------------------------------------------
// 8. Response already committed — returns nil without modifying response
// ---------------------------------------------------------------------------

func TestErrorHandlerMiddleware_ResponseCommitted(t *testing.T) {
	e := echo.New()
	e.Use(ErrorHandlerMiddleware())
	e.GET("/test", func(c echo.Context) error {
		// Write something to commit the response
		c.Response().WriteHeader(http.StatusOK)
		_, _ = c.Response().Write([]byte("already sent"))
		return echo.NewHTTPError(http.StatusInternalServerError, "ignored")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

// ---------------------------------------------------------------------------
// 9. Context canceled — returns nil without rendering
// ---------------------------------------------------------------------------

func TestErrorHandlerMiddleware_ContextCanceled(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("HX-Request", "true")

	// Create a cancelable context and cancel it before the handler runs
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := ErrorHandlerMiddleware()
	handler := mw(func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	})

	err := handler(c)

	// handleError detects context.Canceled and returns nil
	require.NoError(t, err)
	// Body should be empty since nothing was rendered
	require.Empty(t, rec.Body.String(),
		fmt.Sprintf("expected empty body when context canceled, got: %s", rec.Body.String()),
	)
}
