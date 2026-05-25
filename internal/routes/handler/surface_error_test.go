package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corecomponents "catgoose/dothog/web/components/core"
	"github.com/a-h/templ"
	"github.com/catgoose/linkwell"
	"github.com/catgoose/promolog"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPErrorHandler_SurfaceError_PageRendersInsideChrome pins the runtime
// page-surface path: a handler that returns a SurfacePage SurfaceError flows
// through the central error handler and renders inside the host AppNavLayout,
// not the standalone document shell.
func TestHTTPErrorHandler_SurfaceError_PageRendersInsideChrome(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/test", func(c echo.Context) error {
		return NewSurfaceError(c, corecomponents.SurfacePage, http.StatusNotFound,
			"Not Found", "Resource gone.", nil)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	body := strings.ToLower(rec.Body.String())
	assert.Contains(t, body, "app-shell",
		"page surface must compose into the host chrome via the central handler")
	assert.Contains(t, body, "not found")
}

// TestHTTPErrorHandler_SurfaceError_PageBodyOverrideHonored pins the
// SurfaceError.Body escape hatch: routes can supply richer in-chrome content
// (NotFoundPage's resource grid, custom error pages) without bypassing the
// central pipeline. The override replaces the default RenderError(p) while
// still composing inside AppNavLayout for SurfacePage.
func TestHTTPErrorHandler_SurfaceError_PageBodyOverrideHonored(t *testing.T) {
	e := setupEcho(nil)
	const marker = "custom-not-found-marker-abc123"
	e.GET("/test", func(c echo.Context) error {
		return NewSurfaceError(c, corecomponents.SurfacePage, http.StatusNotFound,
			"Not Found", "Resource gone.", nil).
			WithBody(templ.Raw("<div>" + marker + "</div>"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, marker, "Body override must replace the default RenderError content")
	assert.Contains(t, strings.ToLower(body), "app-shell",
		"Body override must still compose inside the host chrome for SurfacePage")
}

// TestHTTPErrorHandler_SurfaceError_DocumentStillStandalone pins the runtime
// document-surface path against the page-surface change: a SurfaceDocument
// SurfaceError must still produce a standalone shell, not get pulled into
// host chrome.
func TestHTTPErrorHandler_SurfaceError_DocumentStillStandalone(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/test", func(c echo.Context) error {
		return NewSurfaceError(c, corecomponents.SurfaceDocument, http.StatusServiceUnavailable,
			"Service Unavailable", "Upstream down.", nil)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	body := strings.ToLower(rec.Body.String())
	assert.Contains(t, body, "<!doctype html>", "document surface owns its own shell")
	assert.NotContains(t, body, "app-shell", "document surface must not pull host chrome in")
	assert.Contains(t, body, "service unavailable")
}

// TestHTTPErrorHandler_SurfaceError_HTMXFallsBackToBanner pins the HTMX
// branch: regardless of the caller-chosen surface, an HTMX request gets the
// banner OOB swap because the central late hook can't safely target arbitrary
// client DOM.
func TestHTTPErrorHandler_SurfaceError_HTMXFallsBackToBanner(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/test", func(c echo.Context) error {
		return NewSurfaceError(c, corecomponents.SurfacePage, http.StatusForbidden,
			"Forbidden", "Access denied.", errors.New("policy: viewer < admin"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	body := strings.ToLower(rec.Body.String())
	assert.NotContains(t, body, "<!doctype html>", "HTMX response is a fragment, not a full document")
	assert.Contains(t, body, "forbidden")
}

// TestHTTPErrorHandler_SurfaceError_500AppendsReportIssue pins the
// request-ID/report-issue contract that the pre-refactor central error path
// preserved for 500+ responses: when the caller doesn't supply explicit
// controls, the default control set must include a Report Issue button so
// the user can attach the request trace.
func TestHTTPErrorHandler_SurfaceError_500AppendsReportIssue(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/test", func(c echo.Context) error {
		return NewSurfaceError(c, corecomponents.SurfaceDocument, http.StatusInternalServerError,
			"Operation failed", "Unexpected database error.", errors.New("pq: deadlock detected"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, linkwell.LabelReportIssue,
		"500 SurfaceError with default controls must surface the Report Issue affordance")
	assert.Contains(t, strings.ToLower(body), "request id",
		"500 SurfaceError must render the request ID so the user has something to report")
}

// TestNewSurfaceError_500DefaultControlsIncludeReport guards the constructor
// contract directly: the default control set for 500+ statuses ends with a
// ReportIssueButton bound to the current request ID. Caller-supplied controls
// override the default and are not augmented (matches the pre-refactor
// HandleHypermediaError behavior).
func TestNewSurfaceError_500DefaultControlsIncludeReport(t *testing.T) {
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	e.GET("/test", func(c echo.Context) error {
		se := NewSurfaceError(c, corecomponents.SurfaceDocument, http.StatusInternalServerError,
			"boom", "", nil)
		require.NotEmpty(t, se.EC.Controls, "500 SurfaceError must seed default controls")
		last := se.EC.Controls[len(se.EC.Controls)-1]
		assert.Equal(t, linkwell.ControlKindReport, last.Kind,
			"500 default control set must end with the Report Issue button")
		assert.NotEmpty(t, se.EC.RequestID,
			"SurfaceError must stamp the correlated request ID so Report Issue can attach the trace")
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

// TestHTTPErrorHandler_SurfaceError_HTMXNegotiatesAcceptedInline pins the
// capability-driven HTMX path: a route that picks SurfaceInline against a
// client that advertised inline acceptance renders the inline panel (not the
// banner OOB swap). The server stays free of DOM-target knowledge — the
// client's hx-target places the fragment.
func TestHTTPErrorHandler_SurfaceError_HTMXNegotiatesAcceptedInline(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/test", func(c echo.Context) error {
		return NewSurfaceError(c, corecomponents.SurfaceInline, http.StatusUnprocessableEntity,
			"Validation failed", "Name is required.", nil)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("HX-Request", "true")
	req.Header.Set(HeaderErrorAcceptSurfaces, "inline,banner")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "alert-error", "negotiated inline render must use the InlineErrorPanel shape")
	assert.Contains(t, body, "Validation failed")
	assert.NotContains(t, body, "hx-swap-oob",
		"inline render must not piggyback the global banner OOB swap")
}

// TestHTTPErrorHandler_SurfaceError_HTMXFallbackKicksIn pins the downgrade
// path: when the chosen surface isn't accepted and a fallback is declared,
// negotiation lands on the fallback rather than silently dropping the error.
func TestHTTPErrorHandler_SurfaceError_HTMXFallbackKicksIn(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/test", func(c echo.Context) error {
		return NewSurfaceError(c, corecomponents.SurfaceInlineFull, http.StatusForbidden,
			"Forbidden", "Access denied.", nil)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("HX-Request", "true")
	req.Header.Set(HeaderErrorAcceptSurfaces, "banner")
	req.Header.Set(HeaderErrorFallbackSurface, "banner")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "hx-swap-oob",
		"banner fallback rides the OOB swap into the global error-status target")
	assert.Contains(t, body, "Forbidden")
}

// TestHTTPErrorHandler_SurfaceError_HTMXNoAdvertisementKeepsBanner pins the
// rule that HTMX clients which advertise no capability headers fall through
// to the banner OOB swap regardless of the route's chosen surface.
func TestHTTPErrorHandler_SurfaceError_HTMXNoAdvertisementKeepsBanner(t *testing.T) {
	e := setupEcho(nil)
	e.GET("/test", func(c echo.Context) error {
		return NewSurfaceError(c, corecomponents.SurfaceInline, http.StatusUnprocessableEntity,
			"Validation failed", "Name is required.", nil)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "hx-swap-oob",
		"without capability advertisement the central path keeps the banner OOB swap")
}

// TestHTTPErrorHandler_SurfaceError_PromotesTrace pins the trace-promotion
// invariant: SurfaceError-based renders still feed the per-request log buffer
// into the shared store, so Report Issue keeps working.
func TestHTTPErrorHandler_SurfaceError_PromotesTrace(t *testing.T) {
	store := &recordingStore{}
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	e.HTTPErrorHandler = NewHTTPErrorHandler(store)
	e.GET("/test", func(c echo.Context) error {
		return NewSurfaceError(c, corecomponents.SurfacePage, http.StatusNotFound,
			"Not Found", "Resource gone.", nil)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	traces := store.snapshot()
	require.Len(t, traces, 1, "SurfaceError must still promote a trace through the central path")
	require.Equal(t, http.StatusNotFound, traces[0].StatusCode)
	require.Equal(t, "/test", traces[0].Route)
	require.NotEmpty(t, traces[0].RequestID)
}
