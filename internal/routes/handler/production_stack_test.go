package handler

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"catgoose/dothog/internal/routes/middleware"

	"github.com/CAFxX/httpcompression"
	"github.com/catgoose/dorman"
	"github.com/catgoose/promolog"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupProductionStack builds the smallest slice of the InitEcho chain needed
// to exercise the compression/error-handler hot path: correlation, security
// headers, raw-writer save, httpcompression, and the custom HTTPErrorHandler.
// The full response-policy bundle is covered by internal/responsepolicy's own
// tests, and auth/CSRF/session do not affect the compressed-writer restore
// path this stack is guarding, so both are deliberately omitted here.
func setupProductionStack(t *testing.T) *echo.Echo {
	t.Helper()
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	e.Use(echo.WrapMiddleware(dorman.SecurityHeaders()))
	e.Use(middleware.RawWriterMiddleware())
	compress, err := httpcompression.DefaultAdapter()
	require.NoError(t, err)
	e.Use(echo.WrapMiddleware(compress))
	e.HTTPErrorHandler = NewHTTPErrorHandler(nil)
	return e
}

func TestProductionStack_SecurityHeadersPresent(t *testing.T) {
	e := setupProductionStack(t)
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "SAMEORIGIN", rec.Header().Get("X-Frame-Options"))
}

func TestProductionStack_RequestIDHeader(t *testing.T) {
	e := setupProductionStack(t)
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	id := rec.Header().Get("X-Request-ID")
	assert.NotEmpty(t, id)
	assert.Len(t, id, 32)
}

func TestProductionStack_GzipCompressesSuccessResponse(t *testing.T) {
	e := setupProductionStack(t)
	e.GET("/test", func(c echo.Context) error {
		// Pad past the small-response threshold so compression actually engages.
		body := strings.Repeat("Hello, this is a test response that should be compressed. ", 20)
		return c.String(http.StatusOK, body)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

	gz, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	defer func() { _ = gz.Close() }()
	body, err := io.ReadAll(gz)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Hello, this is a test response")
}

// TestProductionStack_ErrorHandlerSurvivesCompressedHotPath verifies the
// RawWriter + httpcompression + NewHTTPErrorHandler interaction. The error
// handler runs after httpcompression has finalized its writer, so it must
// restore the raw writer before rendering the fallback 500 page.
func TestProductionStack_ErrorHandlerSurvivesCompressedHotPath(t *testing.T) {
	e := setupProductionStack(t)
	e.GET("/boom", func(c echo.Context) error {
		return errors.New("intentional failure")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	require.NotPanics(t, func() {
		e.ServeHTTP(rec, req)
	}, "error handler must not panic when the compression writer has been finalized")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	body := decodeResponseBody(t, rec)
	assert.Contains(t, body, "operation failed", "fallback 500 message should render")
	assert.Contains(t, body, "<!doctype html>", "non-HTMX errors render the full page")
}

// TestProductionStack_ErrorHandlerHTMXOnCompressedHotPath verifies the HTMX
// branch over the same compressed error path.
func TestProductionStack_ErrorHandlerHTMXOnCompressedHotPath(t *testing.T) {
	e := setupProductionStack(t)
	e.GET("/boom", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad input")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	require.NotPanics(t, func() {
		e.ServeHTTP(rec, req)
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	body := decodeResponseBody(t, rec)
	assert.Contains(t, body, "bad input", "HTMX error response should carry the upstream message")
}

// decodeResponseBody returns the response body as a string, transparently
// gunzipping when Content-Encoding: gzip is set.
func decodeResponseBody(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	if rec.Header().Get("Content-Encoding") != "gzip" {
		return rec.Body.String()
	}
	gz, err := gzip.NewReader(bytes.NewReader(rec.Body.Bytes()))
	require.NoError(t, err, "response advertised gzip encoding but framing was invalid")
	defer func() { _ = gz.Close() }()
	body, err := io.ReadAll(gz)
	require.NoError(t, err)
	return string(body)
}
