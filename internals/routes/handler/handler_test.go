package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"catgoose/harmony/internals/logger"
	"catgoose/harmony/internals/routes/hypermedia"
	"catgoose/harmony/internals/routes/middleware"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type badComponent struct{}

func (badComponent) Render(ctx context.Context, w io.Writer) error {
	return errors.New("render failed")
}

var _ templ.Component = badComponent{}

func init() {
	os.Setenv("LOG_LEVEL", "ERROR")
	logger.Init()
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
	e.Use(middleware.RequestIDMiddleware())
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
	e.Use(middleware.RequestIDMiddleware())
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
	e.Use(middleware.RequestIDMiddleware())
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

func TestDefaultControls_BadRequest(t *testing.T) {
	controls := defaultControls(http.StatusBadRequest, "test-req-id")
	require.Len(t, controls, 2)
	assert.Equal(t, hypermedia.ControlKindDismiss, controls[0].Kind)
	assert.Equal(t, hypermedia.ControlKindReport, controls[1].Kind)
}

func TestDefaultControls_NotFound(t *testing.T) {
	controls := defaultControls(http.StatusNotFound, "test-req-id")
	require.Len(t, controls, 3)
	assert.Equal(t, hypermedia.ControlKindBack, controls[0].Kind)
	assert.Equal(t, hypermedia.ControlKindHTMX, controls[1].Kind)
	assert.Equal(t, hypermedia.ControlKindReport, controls[2].Kind)
}

func TestDefaultControls_Unauthorized(t *testing.T) {
	controls := defaultControls(http.StatusUnauthorized, "test-req-id")
	require.Len(t, controls, 3)
	assert.Equal(t, hypermedia.ControlKindLink, controls[0].Kind)
	assert.Equal(t, "/login", controls[0].Href)
	assert.Equal(t, hypermedia.ControlKindReport, controls[2].Kind)
}

func TestDefaultControls_ServerError(t *testing.T) {
	controls := defaultControls(http.StatusInternalServerError, "test-req-id")
	require.Len(t, controls, 3)
	assert.Equal(t, hypermedia.ControlKindDismiss, controls[0].Kind)
	assert.Equal(t, hypermedia.ControlKindHTMX, controls[1].Kind)
	assert.Equal(t, hypermedia.ControlKindReport, controls[2].Kind)
}

func TestDefaultControls_ExplicitControlsOverride(t *testing.T) {
	c, _ := newEchoContext(http.MethodGet, "/test", nil)
	e := echo.New()
	e.Use(middleware.RequestIDMiddleware())
	c = e.NewContext(c.Request(), c.Response().Writer.(*httptest.ResponseRecorder))

	custom := hypermedia.RetryButton("Try Again", hypermedia.HxMethodGet, "/retry", "#target")
	err := HandleHypermediaError(c, 500, "fail", errors.New("test"), custom)
	require.Error(t, err)

	var hhe *hypermedia.HTTPError
	require.True(t, errors.As(err, &hhe))
	require.Len(t, hhe.EC.Controls, 1)
	assert.Equal(t, "Try Again", hhe.EC.Controls[0].Label)
}

func TestHandleComponent(t *testing.T) {
	c, rec := newEchoContext(http.MethodGet, "/", nil)
	e := echo.New()
	e.Use(middleware.RequestIDMiddleware())
	c = e.NewContext(c.Request(), rec)

	cmp := templ.Raw("<span>content</span>")
	handler := HandleComponent(cmp)
	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Harmony")
	assert.Contains(t, rec.Body.String(), "<span>content</span>")
}
