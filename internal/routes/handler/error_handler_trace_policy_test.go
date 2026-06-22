package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/catgoose/promolog"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// forceTraceEnv drives the promotion policy's environment inputs for one test
// without mutating global env state.
func forceTraceEnv(t *testing.T, isTest, capture bool) {
	t.Helper()
	prevTest, prevCapture := envIsTest, captureTest4xx
	envIsTest = func() bool { return isTest }
	captureTest4xx = func() bool { return capture }
	t.Cleanup(func() { envIsTest, captureTest4xx = prevTest, prevCapture })
}

func newEchoForStatus(store promolog.Storer, route string, status int) *echo.Echo {
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	e.HTTPErrorHandler = NewHTTPErrorHandler(store)
	e.GET(route, func(_ echo.Context) error { return echo.NewHTTPError(status, http.StatusText(status)) })
	return e
}

func serve(e *echo.Echo, method, target string, htmx bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, http.NoBody)
	if htmx {
		req.Header.Set("HX-Request", "true")
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestShouldPromoteTrace(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		isTest  bool
		capture bool
		want    bool
	}{
		{"5xx promotes in test", 500, true, false, true},
		{"5xx promotes outside test", 503, false, false, true},
		{"4xx promotes outside test", 404, false, false, true},
		{"4xx suppressed in test", 404, true, false, false},
		{"4xx promotes in test with opt-in", 404, true, true, true},
		{"422 suppressed in test", 422, true, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, shouldPromoteTrace(tc.status, tc.isTest, tc.capture))
		})
	}
}

func TestEnvCaptureTest4xx(t *testing.T) {
	cases := map[string]bool{
		"true": true, "TRUE": true, "1": true, " true ": true,
		"": false, "false": false, "0": false, "yes": false,
	}
	for value, want := range cases {
		t.Run(value, func(t *testing.T) {
			t.Setenv("ERROR_TRACE_CAPTURE_TEST_4XX", value)
			assert.Equal(t, want, envCaptureTest4xx())
		})
	}
}

func TestTracePromotion_TestMode4xxSuppressed(t *testing.T) {
	forceTraceEnv(t, true, false)
	store := &recordingStore{}
	rec := serve(newEchoWithPromotion(store), http.MethodGet, "/missing", false)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Empty(t, store.snapshot(), "ENV=test 4xx must not promote a trace")
}

func TestTracePromotion_TestMode5xxStillPromotes(t *testing.T) {
	forceTraceEnv(t, true, false)
	store := &recordingStore{}
	rec := serve(newEchoForStatus(store, "/boom", http.StatusInternalServerError), http.MethodGet, "/boom", false)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Len(t, store.snapshot(), 1, "5xx always promotes, even in ENV=test")
}

func TestTracePromotion_NonTest4xxPromotes(t *testing.T) {
	forceTraceEnv(t, false, false)
	store := &recordingStore{}
	rec := serve(newEchoWithPromotion(store), http.MethodGet, "/missing", false)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Len(t, store.snapshot(), 1, "4xx outside ENV=test still promotes")
}

func TestTracePromotion_OptInRestoresTestMode4xx(t *testing.T) {
	forceTraceEnv(t, true, true)
	store := &recordingStore{}
	rec := serve(newEchoWithPromotion(store), http.MethodGet, "/missing", false)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Len(t, store.snapshot(), 1, "ERROR_TRACE_CAPTURE_TEST_4XX re-enables test-mode 4xx promotion")
}

func TestTraceSuppression_NonTest4xxShowsRequestID(t *testing.T) {
	forceTraceEnv(t, false, false)
	rec := serve(newEchoForStatus(&recordingStore{}, "/bad", http.StatusBadRequest), http.MethodGet, "/bad", false)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Request ID", "a promoted 4xx surfaces its trace reference")
}

func TestTraceSuppression_TestMode4xxOmitsRequestID(t *testing.T) {
	forceTraceEnv(t, true, false)
	rec := serve(newEchoForStatus(&recordingStore{}, "/bad", http.StatusBadRequest), http.MethodGet, "/bad", false)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.NotContains(t, rec.Body.String(), "Request ID", "a suppressed 4xx must not surface a dead trace reference")
}

func TestTraceSuppression_5xxKeepsReportAffordance(t *testing.T) {
	forceTraceEnv(t, true, false)
	rec := serve(newEchoForStatus(&recordingStore{}, "/boom", http.StatusInternalServerError), http.MethodGet, "/boom", false)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Request ID", "a reportable 5xx still surfaces its trace reference")
	assert.Contains(t, body, "Report Issue", "a reportable 5xx still offers Report Issue")
}
