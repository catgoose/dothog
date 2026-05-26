package responsepolicy

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/catgoose/dorman"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCtx(method, path string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// TestServerTiming_HeaderFormatOnCommittedResponse pins the wire shape on a
// real (committed) response so DevTools / synthetic monitors keep parsing
// the header. ServerTiming now hooks Response.Before, so it fires when the
// inner handler writes; nil-return handlers don't trigger a write and so
// (correctly) don't emit the header.
func TestServerTiming_HeaderFormatOnCommittedResponse(t *testing.T) {
	c, rec := newCtx(http.MethodGet, "/")
	h := ServerTiming()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	require.NoError(t, h(c))
	hdr := rec.Header().Get("Server-Timing")
	require.NotEmpty(t, hdr)
	assert.Regexp(t, regexp.MustCompile(`^total;dur=\d+;desc="Total"$`), hdr)
}

// TestServerTiming_PresentOnHandlerWriteError covers the case where the
// handler writes a partial response before returning an error. The Before
// hook still runs on the write, so timing reaches the client even when the
// downstream handler ultimately errors.
func TestServerTiming_PresentOnHandlerWriteError(t *testing.T) {
	c, rec := newCtx(http.MethodGet, "/boom")
	want := errors.New("handler failed mid-write")
	h := ServerTiming()(func(c echo.Context) error {
		_ = c.String(http.StatusOK, "partial")
		return want
	})
	assert.ErrorIs(t, h(c), want)
	assert.NotEmpty(t, rec.Header().Get("Server-Timing"))
}

// TestVary_AppendsRatherThanClobbers pins the invariant that the Vary
// middleware appends to whatever Vary values upstream middleware already
// declared, rather than replacing them.
func TestVary_AppendsRatherThanClobbers(t *testing.T) {
	c, _ := newCtx(http.MethodGet, "/")
	c.Response().Header().Add("Vary", "Accept-Encoding")
	h := Vary("HX-Request")(func(c echo.Context) error { return nil })
	require.NoError(t, h(c))
	values := c.Response().Header().Values("Vary")
	assert.Contains(t, values, "Accept-Encoding", "must not clobber existing Vary")
	assert.Contains(t, values, "HX-Request")
}

// TestPreload_FallbackEmitsLinkHeader covers the behind-proxy path where 103
// hints are unsafe — the middleware must still expose the preload set as Link
// headers on the final response so the browser learns about them.
func TestPreload_FallbackEmitsLinkHeader(t *testing.T) {
	c, rec := newCtx(http.MethodGet, "/")
	links := []string{
		"</public/css/tailwind.css>; rel=preload; as=style",
		"</public/js/vendor/htmx.min.js>; rel=preload; as=script",
	}
	h := Preload(links, true)(func(c echo.Context) error { return nil })
	require.NoError(t, h(c))
	got := rec.Header().Values("Link")
	assert.ElementsMatch(t, links, got)
}

// TestInstall_RegistersServerTimingAndVary verifies that Install wires the
// expected response-policy middleware onto a real request path.
func TestInstall_RegistersServerTimingAndVary(t *testing.T) {
	e := echo.New()
	Install(e, Config{})
	e.GET("/test", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Server-Timing"))
	assert.Contains(t, rec.Header().Values("Vary"), "HX-Request")
}

// TestInstall_CSPEmittedWhenConfigured verifies that a configured
// ContentSecurityPolicy is emitted as the Content-Security-Policy header via
// the responsepolicy + dorman path.
func TestInstall_CSPEmittedWhenConfigured(t *testing.T) {
	e := echo.New()
	policy := "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'"
	Install(e, Config{
		Security: dorman.SecurityHeadersConfig{ContentSecurityPolicy: policy},
	})
	e.GET("/test", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, policy, rec.Header().Get("Content-Security-Policy"))
}

// TestInstall_CSPOmittedByDefault keeps the CSP-unselected contract honest:
// the default Install call (no csp feature) emits no CSP header at all, so
// derived apps that have not opted in stay free of inline-handler breakage.
func TestInstall_CSPOmittedByDefault(t *testing.T) {
	e := echo.New()
	Install(e, Config{})
	e.GET("/test", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Content-Security-Policy"))
}
