package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/catgoose/linkwell"

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

	var he *linkwell.HTTPError
	require.ErrorAs(t, err, &he)
	require.Equal(t, http.StatusBadRequest, he.EC.StatusCode)
	require.Equal(t, "invalid input", he.EC.Message)
}

func TestUnauthorized(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/test-path")

	err := Unauthorized(c, "login required")

	var he *linkwell.HTTPError
	require.ErrorAs(t, err, &he)
	require.Equal(t, http.StatusUnauthorized, he.EC.StatusCode)
	require.Equal(t, "login required", he.EC.Message)
}

func TestForbidden(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/test-path")

	err := Forbidden(c, "access denied")

	var he *linkwell.HTTPError
	require.ErrorAs(t, err, &he)
	require.Equal(t, http.StatusForbidden, he.EC.StatusCode)
	require.Equal(t, "access denied", he.EC.Message)
}

func TestNotFound(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/test-path")

	err := NotFound(c, "resource missing")

	var he *linkwell.HTTPError
	require.ErrorAs(t, err, &he)
	require.Equal(t, http.StatusNotFound, he.EC.StatusCode)
	require.Equal(t, "resource missing", he.EC.Message)
}

func TestInternalServerError(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/test-path")

	err := InternalServerError(c, "something broke")

	var he *linkwell.HTTPError
	require.ErrorAs(t, err, &he)
	require.Equal(t, http.StatusInternalServerError, he.EC.StatusCode)
	require.Equal(t, "something broke", he.EC.Message)
}

func TestServiceUnavailable(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/test-path")

	err := ServiceUnavailable(c, "try again later")

	var he *linkwell.HTTPError
	require.ErrorAs(t, err, &he)
	require.Equal(t, http.StatusServiceUnavailable, he.EC.StatusCode)
	require.Equal(t, "try again later", he.EC.Message)
}

func TestHypermediaError(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/items/42")

	ctrl := linkwell.BackButton("Go back")
	ec := HypermediaError(c, http.StatusNotFound, "item not found", nil, ctrl)

	require.Equal(t, http.StatusNotFound, ec.StatusCode)
	require.Equal(t, "item not found", ec.Message)
	require.Equal(t, "/items/42", ec.Route)
	require.True(t, ec.Closable)
	require.Len(t, ec.Controls, 1)
	require.Equal(t, linkwell.ControlKindBack, ec.Controls[0].Kind)
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

	ctrls := []linkwell.Control{
		linkwell.BackButton("Back"),
		linkwell.GoHomeButton("Home", "/", "#main"),
	}
	ec := HypermediaError(c, http.StatusForbidden, "denied", nil, ctrls...)

	require.Len(t, ec.Controls, 2)
	require.Equal(t, linkwell.ControlKindBack, ec.Controls[0].Kind)
	require.Equal(t, linkwell.ControlKindHome, ec.Controls[1].Kind)
}

