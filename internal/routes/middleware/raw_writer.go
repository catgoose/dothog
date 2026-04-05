package middleware

import "github.com/labstack/echo/v4"

// rawWriterKey is the echo context key used to store the original
// http.ResponseWriter before the compression middleware wraps it.
const rawWriterKey = "raw_response_writer"

// RawWriterMiddleware saves the original http.ResponseWriter in the echo
// context so the error handler can bypass the httpcompression writer after
// it has been finalized (closed). Register this middleware immediately
// before the compression middleware.
func RawWriterMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(rawWriterKey, c.Response().Writer)
			return next(c)
		}
	}
}
