package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// rawWriterKey is the echo context key used to store the original
// http.ResponseWriter before the compression middleware wraps it.
const rawWriterKey = "raw_response_writer"

// RawWriterMiddleware stashes the original http.ResponseWriter under rawWriterKey so the
// error handler can bypass a finalised httpcompression writer; register immediately
// before the compression middleware.
func RawWriterMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(rawWriterKey, c.Response().Writer)
			return next(c)
		}
	}
}

// RestoreRawWriter swaps c.Response().Writer back to the raw writer saved by
// RawWriterMiddleware. Call from late hooks (error handlers) that must write
// after httpcompression has finalised its wrapper.
func RestoreRawWriter(c echo.Context) {
	if rw, ok := c.Get(rawWriterKey).(http.ResponseWriter); ok {
		c.Response().Writer = rw
	}
}
