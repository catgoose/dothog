package middleware

import (
	"errors"
	"net/http"
	"regexp"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerTimingMiddleware_HeaderPresent(t *testing.T) {
	c, rec := newTestContext(http.MethodGet, "/")

	mw := ServerTimingMiddleware()
	handler := mw(func(c echo.Context) error {
		return nil
	})
	err := handler(c)

	require.NoError(t, err)
	header := rec.Header().Get("Server-Timing")
	assert.NotEmpty(t, header)
}

func TestServerTimingMiddleware_HeaderFormat(t *testing.T) {
	c, rec := newTestContext(http.MethodGet, "/test")

	mw := ServerTimingMiddleware()
	handler := mw(func(c echo.Context) error {
		return nil
	})
	err := handler(c)

	require.NoError(t, err)
	header := rec.Header().Get("Server-Timing")
	pattern := regexp.MustCompile(`^total;dur=\d+;desc="Total"$`)
	assert.Regexp(t, pattern, header)
}

func TestServerTimingMiddleware_HeaderPresentOnError(t *testing.T) {
	c, rec := newTestContext(http.MethodGet, "/fail")

	mw := ServerTimingMiddleware()
	handlerErr := errors.New("handler failed")
	handler := mw(func(c echo.Context) error {
		return handlerErr
	})
	err := handler(c)

	assert.ErrorIs(t, err, handlerErr)
	header := rec.Header().Get("Server-Timing")
	assert.NotEmpty(t, header, "Server-Timing header must be present even on error")
}
