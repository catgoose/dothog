package response

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"catgoose/go-htmx-demo/internals/routes/htmx"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// textComponent returns a templ.Component that writes the given string.
func textComponent(s string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

// failComponent returns a templ.Component whose Render always fails.
func failComponent() templ.Component {
	return templ.ComponentFunc(func(_ context.Context, _ io.Writer) error {
		return errors.New("render failed")
	})
}

// newTestContext creates an echo.Context backed by an httptest.ResponseRecorder.
func newTestContext() (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

// ---------------------------------------------------------------------------
// Builder tests
// ---------------------------------------------------------------------------

func TestBuilder_DefaultStatus200(t *testing.T) {
	c, rec := newTestContext()

	err := New(c).Component(textComponent("hello")).Send()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "hello", rec.Body.String())
	require.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestBuilder_CustomStatus(t *testing.T) {
	c, rec := newTestContext()

	err := New(c).Status(http.StatusCreated).Component(textComponent("created")).Send()
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestBuilder_NoComponent(t *testing.T) {
	c, rec := newTestContext()

	err := New(c).Send()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Body.String())
}

func TestBuilder_OOB(t *testing.T) {
	c, rec := newTestContext()

	primary := textComponent("<div>main</div>")
	oob1 := textComponent(`<div id="a" hx-swap-oob="innerHTML">oob-a</div>`)
	oob2 := textComponent(`<div id="b" hx-swap-oob="innerHTML">oob-b</div>`)

	err := New(c).Component(primary).OOB(oob1).OOB(oob2).Send()
	require.NoError(t, err)

	body := rec.Body.String()
	require.Contains(t, body, "<div>main</div>")
	require.Contains(t, body, "oob-a")
	require.Contains(t, body, "oob-b")

	// Verify ordering: primary comes before OOB fragments.
	mainIdx := strings.Index(body, "<div>main</div>")
	oobAIdx := strings.Index(body, "oob-a")
	oobBIdx := strings.Index(body, "oob-b")
	require.Less(t, mainIdx, oobAIdx)
	require.Less(t, oobAIdx, oobBIdx)
}

func TestBuilder_Trigger(t *testing.T) {
	c, rec := newTestContext()

	err := New(c).Trigger(`{"foo":"bar"}`).Send()
	require.NoError(t, err)
	require.Equal(t, `{"foo":"bar"}`, rec.Header().Get(htmx.HeaderTrigger))
}

func TestBuilder_TriggerEvent(t *testing.T) {
	c, rec := newTestContext()

	err := New(c).TriggerEvent("created", map[string]any{"id": 1}).Send()
	require.NoError(t, err)

	hdr := rec.Header().Get(htmx.HeaderTrigger)
	require.NotEmpty(t, hdr)
	require.Contains(t, hdr, `"created"`)
	require.Contains(t, hdr, `"id"`)
}

func TestBuilder_TriggerEvent_MarshalError(t *testing.T) {
	c, rec := newTestContext()

	err := New(c).TriggerEvent("bad", make(chan int)).Send()
	require.Error(t, err)
	require.Contains(t, err.Error(), "TriggerEvent")
	// Atomic guarantee: nothing should be written to response body.
	require.Empty(t, rec.Body.String())
}

func TestBuilder_Redirect(t *testing.T) {
	c, rec := newTestContext()

	err := New(c).Redirect("/login").Send()
	require.NoError(t, err)
	require.Equal(t, "/login", rec.Header().Get(htmx.HeaderRedirect))
}

func TestBuilder_PushURL(t *testing.T) {
	c, rec := newTestContext()

	err := New(c).PushURL("/new-page").Send()
	require.NoError(t, err)
	require.Equal(t, "/new-page", rec.Header().Get(htmx.HeaderPushURL))
}

func TestBuilder_Reswap(t *testing.T) {
	c, rec := newTestContext()

	err := New(c).Reswap(htmx.SwapOuterHTML).Send()
	require.NoError(t, err)
	require.Equal(t, string(htmx.SwapOuterHTML), rec.Header().Get(htmx.HeaderReswap))
}

func TestBuilder_Retarget(t *testing.T) {
	c, rec := newTestContext()

	err := New(c).Retarget("#panel").Send()
	require.NoError(t, err)
	require.Equal(t, "#panel", rec.Header().Get(htmx.HeaderRetarget))
}

func TestBuilder_Refresh(t *testing.T) {
	c, rec := newTestContext()

	err := New(c).Refresh().Send()
	require.NoError(t, err)
	require.Equal(t, "true", rec.Header().Get(htmx.HeaderRefresh))
}

func TestBuilder_RenderError(t *testing.T) {
	c, rec := newTestContext()

	err := New(c).Component(failComponent()).Send()
	require.Error(t, err)
	require.Equal(t, "render failed", err.Error())
	// Atomic guarantee: nothing written on render failure.
	require.Empty(t, rec.Body.String())
}

func TestBuilder_OOBRenderError(t *testing.T) {
	c, rec := newTestContext()

	err := New(c).Component(textComponent("ok")).OOB(failComponent()).Send()
	require.Error(t, err)
	require.Equal(t, "render failed", err.Error())
	// Atomic guarantee: primary rendered to buffer but nothing flushed.
	require.Empty(t, rec.Body.String())
}

func TestBuilder_ChainedHeaders(t *testing.T) {
	c, rec := newTestContext()

	err := New(c).
		PushURL("/items/42").
		TriggerEvent("updated", map[string]any{"id": 42}).
		Reswap(htmx.SwapOuterHTML).
		Send()
	require.NoError(t, err)
	require.Equal(t, "/items/42", rec.Header().Get(htmx.HeaderPushURL))
	require.Equal(t, string(htmx.SwapOuterHTML), rec.Header().Get(htmx.HeaderReswap))
	require.NotEmpty(t, rec.Header().Get(htmx.HeaderTrigger))
}

// ---------------------------------------------------------------------------
// OOB tests
// ---------------------------------------------------------------------------

func TestClearErrorOOB(t *testing.T) {
	placement := ClearErrorOOB()
	require.Equal(t, "#error-status", placement.Target)
	require.Equal(t, "innerHTML", placement.Swap)

	// Render the component and verify the HTML output.
	var buf strings.Builder
	err := placement.Component.Render(context.Background(), &buf)
	require.NoError(t, err)
	require.Equal(t, `<div id="error-status" hx-swap-oob="innerHTML"></div>`, buf.String())
}

func TestOOBPlacement_ToComponent(t *testing.T) {
	cmp := textComponent("test-oob")
	placement := OOBPlacement{
		Component: cmp,
		Target:    "#target",
		Swap:      "outerHTML",
	}
	got := placement.ToComponent()

	// Verify ToComponent returns a component that renders identically.
	var expected, actual strings.Builder
	require.NoError(t, cmp.Render(context.Background(), &expected))
	require.NoError(t, got.Render(context.Background(), &actual))
	require.Equal(t, expected.String(), actual.String())
	require.Equal(t, "test-oob", actual.String())
}
