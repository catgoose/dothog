package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"catgoose/dothog/internal/routes/handler"

	"github.com/catgoose/promolog"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupScenarios builds an Echo wired with correlation + the production
// HTTPErrorHandler so SurfaceError flows through the real central pipeline,
// then registers the scaffold-facing /examples/error-scenarios routes.
func setupScenarios(t *testing.T) *echo.Echo {
	t.Helper()
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	e.HTTPErrorHandler = handler.NewHTTPErrorHandler(nil)
	ar := &AppRoutes{e: e, ctx: context.Background()}
	ar.initErrorScenariosRoutes()
	return e
}

func TestErrorScenarios_Index_Renders(t *testing.T) {
	e := setupScenarios(t)

	req := httptest.NewRequest(http.MethodGet, "/examples/error-scenarios", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Error Scenarios")
	assert.Contains(t, body, "data-error-accept=\"inline,banner\"")
	assert.Contains(t, body, "/examples/error-scenarios/route-404",
		"index must link to the route-404 scenario")
	assert.Contains(t, body, "/examples/error-scenarios/auth-boundary",
		"index must link to the auth-boundary scenario")
}

func TestErrorScenarios_BannerFallback_HTMXNoAdvertisement(t *testing.T) {
	e := setupScenarios(t)
	req := httptest.NewRequest(http.MethodGet, "/examples/error-scenarios/banner-fallback", http.NoBody)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "hx-swap-oob",
		"no capability advertised → central path keeps the banner OOB swap")
	assert.Contains(t, body, "Background job failed")
}

func TestErrorScenarios_ValidationInline_NegotiatesInline(t *testing.T) {
	e := setupScenarios(t)
	req := httptest.NewRequest(http.MethodGet, "/examples/error-scenarios/validation-inline", http.NoBody)
	req.Header.Set("HX-Request", "true")
	req.Header.Set(handler.HeaderErrorAcceptSurfaces, "inline,banner")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "alert-error",
		"inline negotiation must produce InlineErrorPanel shape")
	assert.NotContains(t, body, "hx-swap-oob",
		"negotiated inline render must not piggyback the global banner OOB swap")
	assert.Contains(t, body, "Email is required")
}

func TestErrorScenarios_Route404_PageInChrome(t *testing.T) {
	e := setupScenarios(t)
	req := httptest.NewRequest(http.MethodGet, "/examples/error-scenarios/route-404", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	body := strings.ToLower(rec.Body.String())
	assert.Contains(t, body, "app-shell",
		"page surface must compose inside the host AppNavLayout chrome")
	assert.Contains(t, body, "project not found")
	assert.Contains(t, body, "flex min-h-[70vh] items-center justify-center py-8",
		"page surface must wrap content in the 70vh viewport-centered flex container so it reads as a real route view")
}

func TestErrorScenarios_AuthBoundary_StandaloneShell(t *testing.T) {
	e := setupScenarios(t)
	req := httptest.NewRequest(http.MethodGet, "/examples/error-scenarios/auth-boundary", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	body := strings.ToLower(rec.Body.String())
	assert.Contains(t, body, "<!doctype html>",
		"document surface emits its own standalone shell")
	assert.NotContains(t, body, "app-shell",
		"document surface must not pull host chrome in")
	assert.Contains(t, body, "sign in required")
	assert.NotContains(t, body, "report issue",
		"auth-boundary scenario must not include a Report Issue control")
}

func TestErrorScenarios_Reportable500_AppendsReportIssue(t *testing.T) {
	e := setupScenarios(t)
	req := httptest.NewRequest(http.MethodGet, "/examples/error-scenarios/reportable-500", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Report Issue",
		"500 default controls must surface the Report Issue affordance through the scenario")
	assert.Contains(t, strings.ToLower(body), "request id",
		"500 render must include the request ID the report flow attaches")
}
