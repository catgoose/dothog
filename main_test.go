package main

import (
	"catgoose/go-htmx-template/internals/config"
	"catgoose/go-htmx-template/internals/logger"
	"catgoose/go-htmx-template/internals/routes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/catgoose/dio"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAppEcho(t *testing.T) *echo.Echo {
	t.Helper()
	config.ResetForTesting()
	os.Setenv("SERVER_LISTEN_PORT", "0")
	os.Setenv("LOG_LEVEL", "ERROR")
	t.Cleanup(func() {
		os.Unsetenv("SERVER_LISTEN_PORT")
		os.Unsetenv("LOG_LEVEL")
	})

	require.NoError(t, dio.InitEnvironment(nil))
	logger.Init()
	cfg, err := config.GetConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	ctx := context.Background()
	e, err := routes.InitEcho(ctx, staticFS, cfg)
	require.NoError(t, err)
	require.NotNil(t, e)

	ar := routes.NewAppRoutes(ctx, e)
	require.NoError(t, ar.InitRoutes())
	return e
}

func TestApplicationStartup(t *testing.T) {
	// Set up test environment
	os.Setenv("SERVER_LISTEN_PORT", "0")
	defer os.Unsetenv("SERVER_LISTEN_PORT")

	// Test that we can initialize the application components
	// This tests the startup sequence without actually starting the server

	// Test environment initialization
	err := dio.InitEnvironment(nil)
	require.NoError(t, err)

	cfg, err := config.GetConfig()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	e, err := routes.InitEcho(context.Background(), staticFS, cfg)
	require.NoError(t, err)
	assert.NotNil(t, e)

	// Test context creation
	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	assert.NotNil(t, appCtx)

	// Test routes setup
	ar := routes.NewAppRoutes(appCtx, e)
	assert.NotNil(t, ar)

	err = ar.InitRoutes()
	require.NoError(t, err)
}

func TestContextCancellation(t *testing.T) {
	// Test that context cancellation works properly
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context
	cancel()

	// Check that context is done
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("Context should be done after cancellation")
	}
}

func TestGracefulShutdown(t *testing.T) {
	// Test that we can create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()

	// Context should not be done immediately
	select {
	case <-shutdownCtx.Done():
		t.Error("Shutdown context should not be done immediately")
	default:
		// Expected
	}

	// Wait for timeout
	time.Sleep(1100 * time.Millisecond)

	// Context should be done after timeout
	select {
	case <-shutdownCtx.Done():
		// Expected
	default:
		t.Error("Shutdown context should be done after timeout")
	}
}

func TestEnvironmentVariables(t *testing.T) {
	// Reset config singleton to ensure clean test
	config.ResetForTesting()

	// Test environment variable handling
	testPort := "12345"
	os.Setenv("SERVER_LISTEN_PORT", testPort)
	defer os.Unsetenv("SERVER_LISTEN_PORT")

	config, err := config.GetConfig()
	require.NoError(t, err)
	assert.Equal(t, testPort, config.ServerPort)
}

func TestLoggerIntegration(t *testing.T) {
	logger.Init()
	log := logger.Get()
	assert.NotNil(t, log)

	assert.NotPanics(t, func() {
		logger.Info("test message")
		logger.Error("test error")
	})
}

func TestWorkflowGETRoot(t *testing.T) {
	e := setupAppEcho(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "HTMX Go Template")
}

func TestWorkflowGETHealth(t *testing.T) {
	e := setupAppEcho(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"ok"`)
}

func TestWorkflowStatic(t *testing.T) {
	e := setupAppEcho(t)
	req := httptest.NewRequest(http.MethodGet, "/public/css/tailwind.css", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWorkflowRequestID(t *testing.T) {
	e := setupAppEcho(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
}

func TestWorkflow404(t *testing.T) {
	e := setupAppEcho(t)
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWorkflow404HTMX(t *testing.T) {
	e := setupAppEcho(t)
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.True(t, strings.Contains(rec.Body.String(), "Page not found") || strings.Contains(rec.Body.String(), "error-message-content"))
}

func TestWorkflowErrorHandler(t *testing.T) {
	e := setupAppEcho(t)
	e.GET("/test-error", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusInternalServerError, "test error message")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-error", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestWorkflowErrorHandlerHTMX(t *testing.T) {
	e := setupAppEcho(t)
	e.GET("/test-error-htmx", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-error-htmx", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "bad request")
}

func BenchmarkServerStartup(b *testing.B) {
	// Benchmark the server startup process
	os.Setenv("SERVER_LISTEN_PORT", "0")
	defer os.Unsetenv("SERVER_LISTEN_PORT")

	for i := 0; i < b.N; i++ {
		cfg, _ := config.GetConfig()
		_, _ = routes.InitEcho(context.Background(), staticFS, cfg)
	}
}
