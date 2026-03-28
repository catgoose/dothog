package middleware

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
)

// ServerTimingMiddleware measures request processing time and emits
// a Server-Timing HTTP header visible in browser DevTools.
func ServerTimingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			dur := time.Since(start).Milliseconds()
			c.Response().Header().Set("Server-Timing",
				fmt.Sprintf("total;dur=%d;desc=\"Total\"", dur))
			return err
		}
	}
}
