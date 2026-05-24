package handler

import (
	"net/http"
	"strings"
	"testing"

	corecomponents "catgoose/dothog/web/components/core"
	"github.com/catgoose/promolog"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderErrorSurface_PageRendersInsideHostChrome pins the Page surface
// to the in-chrome render path: the error sits inside the host AppNavLayout
// (so the app nav surrounds it) rather than emitting its own minimal shell.
func TestRenderErrorSurface_PageRendersInsideHostChrome(t *testing.T) {
	c, rec := newEchoContext(http.MethodGet, "/missing", nil)
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	c = e.NewContext(c.Request(), rec)

	p := corecomponents.NewPageError(http.StatusNotFound, "Not Found",
		"The page you requested does not exist.", "/missing", "req-test-page")
	require.NoError(t, RenderErrorSurface(c, p))

	body := strings.ToLower(rec.Body.String())
	require.NotEmpty(t, body)
	assert.Contains(t, body, "app-shell",
		"page surface must compose into the host AppNavLayout chrome")
	assert.Contains(t, body, "not found")
	assert.Contains(t, body, "/missing")
}

// TestRenderErrorSurface_DocumentEmitsStandaloneShell pins the Document
// surface to the standalone-shell path: the output owns its own minimal
// <html>/<body> (no app nav) so it can render at auth/system boundaries that
// have no chrome to compose into.
func TestRenderErrorSurface_DocumentEmitsStandaloneShell(t *testing.T) {
	c, rec := newEchoContext(http.MethodGet, "/boom", nil)
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	c = e.NewContext(c.Request(), rec)

	p := corecomponents.NewDocumentError(http.StatusServiceUnavailable,
		"Service Unavailable",
		"A downstream dependency is not responding.",
		"/boom", "req-test-doc", "dark")
	require.NoError(t, RenderErrorSurface(c, p))

	body := strings.ToLower(rec.Body.String())
	assert.Contains(t, body, "<!doctype html>",
		"document surface must emit its own shell so it can stand alone")
	assert.NotContains(t, body, "app-shell",
		"document surface must not pull in the host AppNavLayout chrome")
	assert.Contains(t, body, "service unavailable")
	assert.Contains(t, body, "/boom")
}

