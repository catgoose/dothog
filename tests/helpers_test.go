package tests

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// TestServer represents a test server instance
type TestServer struct {
	Echo *echo.Echo
	URL  string
}

// SetupTestServer creates a test server for integration tests
func SetupTestServer(t *testing.T) *TestServer {
	// Set test environment variables
	os.Setenv("SERVER_LISTEN_PORT", "0")
	os.Setenv("LOG_LEVEL", "ERROR") // Reduce log noise in tests

	e := echo.New()

	// Create a test server
	server := httptest.NewServer(e)

	return &TestServer{
		Echo: e,
		URL:  server.URL,
	}
}

// CleanupTestServer cleans up test server resources
func CleanupTestServer(ts *TestServer) {
	if ts != nil && ts.Echo != nil {
		ts.Echo.Close()
	}
}

// MakeTestRequest makes a test request to the server
func MakeTestRequest(t *testing.T, e *echo.Echo, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// AssertResponseStatus asserts that the response has the expected status code
func AssertResponseStatus(t *testing.T, rec *httptest.ResponseRecorder, expectedStatus int) {
	require.Equal(t, expectedStatus, rec.Code, "Expected status %d, got %d", expectedStatus, rec.Code)
}

// AssertResponseBodyContains asserts that the response body contains the expected text
func AssertResponseBodyContains(t *testing.T, rec *httptest.ResponseRecorder, expectedText string) {
	require.Contains(t, rec.Body.String(), expectedText, "Response body should contain '%s'", expectedText)
}

// CreateTestContext creates a test context with optional timeout
func CreateTestContext(t *testing.T, timeout ...time.Duration) context.Context {
	if len(timeout) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout[0])
		t.Cleanup(cancel)
		return ctx
	}
	return context.Background()
}

// SetTestEnvironment sets up test environment variables
func SetTestEnvironment(t *testing.T) {
	os.Setenv("SERVER_LISTEN_PORT", "0")
	os.Setenv("LOG_LEVEL", "ERROR")
	os.Setenv("ENV", "test")
}

// CleanupTestEnvironment cleans up test environment variables
func CleanupTestEnvironment(t *testing.T) {
	os.Unsetenv("SERVER_LISTEN_PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("ENV")
}
