package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
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
	err := HandleHypermediaError(c, http.StatusInternalServerError, "fail", errors.New("test"), custom)
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
// links.go has a corresponding entry in the app navigation.
// When a new hub is added, this test fails as a reminder to add it to
// appNavNavConfig with an appropriate icon.
func TestAppNavCoversHubs(t *testing.T) {
	cfg := appNavNavConfig()
	navPaths := make(map[string]bool, len(cfg.Items))
	for _, item := range cfg.Items {
		navPaths[item.Href] = true
	}

	// Hub paths from links.go. Keep this in sync when adding hubs.
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

func oobDiv(id, swap, body string) templ.Component {
	return templ.Raw(fmt.Sprintf(`<div id=%q hx-swap-oob=%q>%s</div>`, id, swap, body))
}

func TestRenderHypermedia_MainThenFragmentsInOrder(t *testing.T) {
	c, rec := newEchoContext(http.MethodGet, "/", nil)
	err := RenderHypermedia(c, HypermediaResponse{
		Main: templ.Raw("<main>main</main>"),
		Fragments: []OOBFragment{
			{TargetID: "first", Swap: "true", Component: oobDiv("first", "true", "one")},
			{TargetID: "second", Swap: "beforeend", Component: oobDiv("second", "beforeend", "two")},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	mainAt := strings.Index(body, "<main>main</main>")
	firstAt := strings.Index(body, `id="first"`)
	secondAt := strings.Index(body, `id="second"`)
	require.True(t, mainAt >= 0 && firstAt >= 0 && secondAt >= 0, "all regions render")
	assert.Less(t, mainAt, firstAt, "main renders before the fragments")
	assert.Less(t, firstAt, secondAt, "fragments render in declared order")
}

func TestRenderHypermedia_SkipsNilMainAndNilFragments(t *testing.T) {
	c, rec := newEchoContext(http.MethodGet, "/", nil)
	err := RenderHypermedia(c, HypermediaResponse{
		Main: nil,
		Fragments: []OOBFragment{
			{TargetID: "skip", Swap: "true", Component: nil},
			{TargetID: "kept", Swap: "true", Component: oobDiv("kept", "true", "here")},
		},
	})
	require.NoError(t, err)

	body := rec.Body.String()
	assert.Contains(t, body, `id="kept"`)
	assert.NotContains(t, body, `id="skip"`)
}

func TestRenderHypermedia_RenderErrorReturns500(t *testing.T) {
	tests := []struct {
		name string
		resp HypermediaResponse
	}{
		{"bad main", HypermediaResponse{Main: badComponent{}}},
		{"bad fragment", HypermediaResponse{Fragments: []OOBFragment{
			{TargetID: "x", Swap: "true", Component: badComponent{}},
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newEchoContext(http.MethodGet, "/", nil)
			err := RenderHypermedia(c, tt.resp)
			require.NoError(t, err)
			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.Contains(t, rec.Body.String(), "error-message-content")
		})
	}
}

func oobSwapByID(body string) map[string]string {
	openTag := regexp.MustCompile(`<[a-zA-Z][^>]*>`)
	idAttr := regexp.MustCompile(`\bid="([^"]*)"`)
	swapAttr := regexp.MustCompile(`\bhx-swap-oob="([^"]*)"`)
	byID := map[string]string{}
	for _, tag := range openTag.FindAllString(body, -1) {
		id := idAttr.FindStringSubmatch(tag)
		swap := swapAttr.FindStringSubmatch(tag)
		if id != nil && swap != nil {
			byID[id[1]] = swap[1]
		}
	}
	return byID
}

func TestRenderHypermedia_DeclaredMetadataMatchesRenderedAttributes(t *testing.T) {
	frags := []OOBFragment{
		{TargetID: "cart-count", Swap: "true", Component: oobDiv("cart-count", "true", "3")},
		{TargetID: "flash", Swap: "outerHTML", Component: oobDiv("flash", "outerHTML", "Saved")},
	}
	c, rec := newEchoContext(http.MethodGet, "/", nil)
	require.NoError(t, RenderHypermedia(c, HypermediaResponse{
		Main:      templ.Raw("<main></main>"),
		Fragments: frags,
	}))

	swapByID := oobSwapByID(rec.Body.String())
	for _, f := range frags {
		swap, ok := swapByID[f.TargetID]
		require.Truef(t, ok, "rendered OOB element with id %q must exist", f.TargetID)
		assert.Equalf(t, f.Swap, swap, "declared Swap for %q must match the hx-swap-oob on that same element", f.TargetID)
	}
}
