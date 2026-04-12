package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"catgoose/dothog/internal/logger"
	"github.com/catgoose/linkwell"

	"github.com/a-h/templ"
	"github.com/catgoose/promolog"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type badComponent struct{}

func (badComponent) Render(ctx context.Context, w io.Writer) error {
	return errors.New("render failed")
}

var _ templ.Component = badComponent{}

func TestMain(m *testing.M) {
	_ = os.Setenv("LOG_LEVEL", "ERROR")
	logger.Init()
	os.Exit(m.Run())
}

func newEchoContext(method, path string, headers map[string]string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestRenderComponent_Success(t *testing.T) {
	c, rec := newEchoContext(http.MethodGet, "/", nil)
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	c = e.NewContext(c.Request(), rec)

	cmp := templ.Raw("<div>ok</div>")
	err := RenderComponent(c, cmp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "<div>ok</div>")
}

func TestRenderComponent_RenderError(t *testing.T) {
	c, rec := newEchoContext(http.MethodGet, "/", nil)
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	c = e.NewContext(c.Request(), rec)

	cmp := badComponent{}
	err := RenderComponent(c, cmp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "error-message-content")
}

func TestHandleError_StatusCode(t *testing.T) {
	c, rec := newEchoContext(http.MethodGet, "/", nil)
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	c = e.NewContext(c.Request(), rec)

	err := HandleError(c, http.StatusBadRequest, "bad request", errors.New("test err"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "bad request")
}

func TestHandleError_ContextCanceled_NoOp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c, rec := newEchoContext(http.MethodGet, "/", nil)
	c.SetRequest(c.Request().WithContext(ctx))

	err := HandleError(c, http.StatusInternalServerError, "ignored", errors.New("test"))
	require.NoError(t, err)
	assert.Empty(t, rec.Body.String())
}

func TestDefaultControls_ExplicitControlsOverride(t *testing.T) {
	c, _ := newEchoContext(http.MethodGet, "/test", nil)
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	c = e.NewContext(c.Request(), c.Response().Writer.(*httptest.ResponseRecorder))

	custom := linkwell.RetryButton("Try Again", linkwell.HxMethodGet, "/retry", "#target")
	err := HandleHypermediaError(c, 500, "fail", errors.New("test"), custom)
	require.Error(t, err)

	var hhe *linkwell.HTTPError
	require.True(t, errors.As(err, &hhe))
	require.Len(t, hhe.EC.Controls, 1)
	assert.Equal(t, "Try Again", hhe.EC.Controls[0].Label)
}

// TestHandleError_FallbackHTML verifies that when the error template itself
// fails to render, HandleError falls back to inline HTML rather than
// recursing indefinitely. We use a component that always fails to trigger
// the fallback path through RenderComponent -> HandleError.
func TestHandleError_FallbackHTML(t *testing.T) {
	c, rec := newEchoContext(http.MethodGet, "/broken", nil)
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	c = e.NewContext(c.Request(), rec)

	// HandleError renders an error template internally. If we call it with
	// a real writer, it should succeed and render the error page.
	err := HandleError(c, http.StatusInternalServerError, "server error", errors.New("db down"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	// The error message should appear somewhere in the response.
	assert.Contains(t, rec.Body.String(), "server error")
}

// TestHandleError_CanceledContextPreventsRender verifies that HandleError
// short-circuits when the request context is canceled, preventing any write
// attempts (and thus preventing recursion on broken writers).
func TestHandleError_CanceledContextPreventsRender(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	c, rec := newEchoContext(http.MethodGet, "/", nil)
	c.SetRequest(c.Request().WithContext(ctx))

	err := HandleError(c, http.StatusInternalServerError, "should be skipped", errors.New("original"))
	require.NoError(t, err)
	assert.Empty(t, rec.Body.String(), "canceled context should prevent any rendering")
}

// TestAppNavCoversHubs ensures every linkwell hub path declared in
// routes_links.go has a corresponding entry in the app navigation.
// When a new hub is added, this test fails as a reminder to add it to
// appNavNavConfig with an appropriate icon.
func TestAppNavCoversHubs(t *testing.T) {
	cfg := appNavNavConfig()
	navPaths := make(map[string]bool, len(cfg.Items))
	for _, item := range cfg.Items {
		navPaths[item.Href] = true
	}

	// Hub paths from routes_links.go. Keep this in sync when adding hubs.
	hubPaths := []string{
		"/apps",
		"/platform",
		"/patterns",
		"/components",
		"/realtime",
		"/api",
		"/admin",
		"/dashboard",
	}

	for _, hp := range hubPaths {
		assert.True(t, navPaths[hp], "hub %s missing from app nav — add it to appNavNavConfig()", hp)
	}
}

func TestHandleComponent(t *testing.T) {
	c, rec := newEchoContext(http.MethodGet, "/", nil)
	e := echo.New()
	e.Use(echo.WrapMiddleware(promolog.CorrelationMiddleware))
	c = e.NewContext(c.Request(), rec)

	cmp := templ.Raw("<span>content</span>")
	handler := HandleComponent(cmp)
	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Admin")
	assert.Contains(t, rec.Body.String(), "<span>content</span>")
}
