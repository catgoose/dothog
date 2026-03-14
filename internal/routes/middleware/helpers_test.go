package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"catgoose/dothog/internal/routes/hypermedia"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers: fresh echo context per test
// ---------------------------------------------------------------------------

func newTestContext(method, path string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

// ---------------------------------------------------------------------------
// errors.go helpers
// ---------------------------------------------------------------------------

func TestBadRequest(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/test-path")

	err := BadRequest(c, "invalid input")

	var he *echo.HTTPError
	require.ErrorAs(t, err, &he)
	require.Equal(t, http.StatusBadRequest, he.Code)
	require.Equal(t, "invalid input", he.Message)
}

func TestUnauthorized(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/test-path")

	err := Unauthorized(c, "login required")

	var he *echo.HTTPError
	require.ErrorAs(t, err, &he)
	require.Equal(t, http.StatusUnauthorized, he.Code)
	require.Equal(t, "login required", he.Message)
}

func TestForbidden(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/test-path")

	err := Forbidden(c, "access denied")

	var he *echo.HTTPError
	require.ErrorAs(t, err, &he)
	require.Equal(t, http.StatusForbidden, he.Code)
	require.Equal(t, "access denied", he.Message)
}

func TestNotFound(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/test-path")

	err := NotFound(c, "resource missing")

	var he *echo.HTTPError
	require.ErrorAs(t, err, &he)
	require.Equal(t, http.StatusNotFound, he.Code)
	require.Equal(t, "resource missing", he.Message)
}

func TestInternalServerError(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/test-path")

	err := InternalServerError(c, "something broke")

	var he *echo.HTTPError
	require.ErrorAs(t, err, &he)
	require.Equal(t, http.StatusInternalServerError, he.Code)
	require.Equal(t, "something broke", he.Message)
}

func TestServiceUnavailable(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/test-path")

	err := ServiceUnavailable(c, "try again later")

	var he *echo.HTTPError
	require.ErrorAs(t, err, &he)
	require.Equal(t, http.StatusServiceUnavailable, he.Code)
	require.Equal(t, "try again later", he.Message)
}

func TestHypermediaError(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/items/42")

	ctrl := hypermedia.BackButton("Go back")
	ec := HypermediaError(c, http.StatusNotFound, "item not found", nil, ctrl)

	require.Equal(t, http.StatusNotFound, ec.StatusCode)
	require.Equal(t, "item not found", ec.Message)
	require.Equal(t, "/items/42", ec.Route)
	require.True(t, ec.Closable)
	require.Len(t, ec.Controls, 1)
	require.Equal(t, hypermedia.ControlKindBack, ec.Controls[0].Kind)
	require.Equal(t, "Go back", ec.Controls[0].Label)
	require.Nil(t, ec.Err)
	// RequestID is empty because RequestIDMiddleware was not applied
	require.Empty(t, ec.RequestID)
}

func TestHypermediaError_WithWrappedError(t *testing.T) {
	c, _ := newTestContext(http.MethodPost, "/submit")
	wrappedErr := echo.NewHTTPError(http.StatusBadGateway, "upstream failed")

	ec := HypermediaError(c, http.StatusBadGateway, "gateway error", wrappedErr)

	require.Equal(t, http.StatusBadGateway, ec.StatusCode)
	require.Equal(t, "gateway error", ec.Message)
	require.Equal(t, "/submit", ec.Route)
	require.True(t, ec.Closable)
	require.Empty(t, ec.Controls)
	require.ErrorIs(t, ec.Err, wrappedErr)
}

func TestHypermediaError_MultipleControls(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/admin")

	ctrls := []hypermedia.Control{
		hypermedia.BackButton("Back"),
		hypermedia.GoHomeButton("Home", "/", "#main"),
	}
	ec := HypermediaError(c, http.StatusForbidden, "denied", nil, ctrls...)

	require.Len(t, ec.Controls, 2)
	require.Equal(t, hypermedia.ControlKindBack, ec.Controls[0].Kind)
	require.Equal(t, hypermedia.ControlKindHTMX, ec.Controls[1].Kind)
}

// ---------------------------------------------------------------------------
// status.go helpers
// ---------------------------------------------------------------------------

func dummyComponent(html string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, html)
		return err
	})
}

func TestCreated(t *testing.T) {
	c, rec := newTestContext(http.MethodPost, "/items")
	cmp := dummyComponent("<div>created</div>")

	err := Created(c, cmp).Send()

	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Contains(t, rec.Body.String(), "<div>created</div>")
}

func TestAccepted(t *testing.T) {
	c, rec := newTestContext(http.MethodPost, "/jobs")
	cmp := dummyComponent("<span>accepted</span>")

	err := Accepted(c, cmp).Send()

	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Contains(t, rec.Body.String(), "<span>accepted</span>")
}

func TestNoContent(t *testing.T) {
	c, rec := newTestContext(http.MethodDelete, "/items/1")

	err := NoContent(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "none", rec.Header().Get("HX-Reswap"))
}

func TestSeeOther(t *testing.T) {
	c, rec := newTestContext(http.MethodPost, "/login")

	err := SeeOther(c, "/dashboard")

	require.NoError(t, err)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/dashboard", rec.Header().Get("HX-Redirect"))
}

// ---------------------------------------------------------------------------
// Builder chaining: Created/Accepted with extra builder methods
// ---------------------------------------------------------------------------

func TestCreated_WithPushURL(t *testing.T) {
	c, rec := newTestContext(http.MethodPost, "/items")
	cmp := dummyComponent("<tr>new row</tr>")

	err := Created(c, cmp).PushURL("/items/99").Send()

	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "/items/99", rec.Header().Get("HX-Push-Url"))
	require.Contains(t, rec.Body.String(), "<tr>new row</tr>")
}

func TestCreated_WithTriggerEvent(t *testing.T) {
	c, rec := newTestContext(http.MethodPost, "/items")
	cmp := dummyComponent("<tr>row</tr>")

	err := Created(c, cmp).TriggerEvent("item-created", map[string]string{"id": "42"}).Send()

	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Contains(t, rec.Header().Get("HX-Trigger"), "item-created")
}

func TestAccepted_WithTriggerEvent(t *testing.T) {
	c, rec := newTestContext(http.MethodPost, "/jobs")
	cmp := dummyComponent("<div>queued</div>")

	err := Accepted(c, cmp).TriggerEvent("job-queued", nil).Send()

	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Contains(t, rec.Header().Get("HX-Trigger"), "job-queued")
}

func TestSeeOther_DifferentPaths(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"root", "/"},
		{"nested path", "/admin/settings"},
		{"with query", "/search?q=test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newTestContext(http.MethodPost, "/form")

			err := SeeOther(c, tt.url)

			require.NoError(t, err)
			require.Equal(t, http.StatusSeeOther, rec.Code)
			require.Equal(t, tt.url, rec.Header().Get("HX-Redirect"))
		})
	}
}

func TestNoContent_EmptyBody(t *testing.T) {
	c, rec := newTestContext(http.MethodDelete, "/items/1")

	err := NoContent(c)

	require.NoError(t, err)
	require.Empty(t, rec.Body.String())
}
