package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"catgoose/go-htmx-demo/internals/logger"
	"catgoose/go-htmx-demo/internals/routes/middleware"

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
	assert.Contains(t, rec.Body.String(), "HTMX Go Template")
	assert.Contains(t, rec.Body.String(), "<span>content</span>")
}

