package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/catgoose/porter"
	"github.com/catgoose/promolog"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupFullEcho mirrors the middleware stack from InitEcho (minus auth/CSRF/session).
func setupFullEcho() *echo.Echo {
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	e.Use(echo.WrapMiddleware(porter.SecurityHeaders()))
	e.Use(echoMiddleware.Gzip())
	e.HTTPErrorHandler = NewHTTPErrorHandler(nil)
	return e
}

func TestSecureHeaders_Present(t *testing.T) {
	e := setupFullEcho()
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

func TestGzip_CompressesResponse(t *testing.T) {
	e := setupFullEcho()
	e.GET("/test", func(c echo.Context) error {
		// Return enough content for gzip to kick in (small responses may be skipped)
		body := "Hello, this is a test response that should be compressed by gzip middleware."
		for range 10 {
			body += " Padding to ensure the response is large enough for compression."
		}
		return c.String(http.StatusOK, body)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

	// Verify the body is valid gzip
	gz, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	defer func() { _ = gz.Close() }()
	body, err := io.ReadAll(gz)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Hello, this is a test response")
}

func TestRequestID_HeaderPresent(t *testing.T) {
	e := setupFullEcho()
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
