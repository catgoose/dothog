package responsepolicy

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

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

// TestVary_AppendsRatherThanClobbers pins the fix called out by plan-048:
// the old routes.go helper used Set, which discarded any Vary value upstream
// middleware already declared. Add preserves them.
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
		"</public/js/htmx.min.js>; rel=preload; as=script",
	}
	h := Preload(links, true)(func(c echo.Context) error { return nil })
	require.NoError(t, h(c))
	got := rec.Header().Values("Link")
	assert.ElementsMatch(t, links, got)
}

// TestInstall_RegistersServerTimingAndVary wires Install onto a fresh Echo
// and asserts the canonical chain is in effect on a real request, so a
// reorder/regression in the package surface is caught here.
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
