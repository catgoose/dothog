package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"catgoose/dothog/internal/logger"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("LOG_LEVEL", "ERROR")
	logger.Init()
}

func TestRequestIDMiddleware_SetsHeaderAndContext(t *testing.T) {
	e := echo.New()
	e.Use(RequestIDMiddleware())
	e.GET("/", func(c echo.Context) error {
		id := GetRequestID(c)
		assert.NotEmpty(t, id)
		return c.String(http.StatusOK, id)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
	assert.Equal(t, rec.Header().Get("X-Request-ID"), rec.Body.String())
}

func TestRequestIDMiddleware_CallsNext(t *testing.T) {
	e := echo.New()
	nextCalled := false
	e.Use(RequestIDMiddleware())
	e.GET("/", func(c echo.Context) error {
		nextCalled = true
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetRequestID_EmptyWhenNotSet(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	assert.Empty(t, GetRequestID(c))
}

func TestRequestIDMiddleware_LatencyLogged(t *testing.T) {
	e := echo.New()
	handlerDelay := 50 * time.Millisecond
	e.Use(RequestIDMiddleware())
	e.GET("/slow", func(c echo.Context) error {
		time.Sleep(handlerDelay)
		return c.String(http.StatusOK, "done")
	})

	start := time.Now()
	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "done", rec.Body.String())
	// Verify that the middleware didn't short-circuit; elapsed time should be >= handler delay
	require.GreaterOrEqual(t, elapsed, handlerDelay, "request should take at least the handler delay")
}
