package responsepolicy

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
)

// ServerTiming measures the wall-clock duration of the inner handler and
// emits a Server-Timing header so browser DevTools can chart it. The header
// is written via echo.Response.Before so it sticks even on responses that
// commit through c.String / c.JSON / c.Render — the previous "set after
// next" shape was a no-op for every real handler because the body had
// already flushed by the time the outer middleware ran.
func ServerTiming() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			c.Response().Before(func() {
				dur := time.Since(start).Milliseconds()
				c.Response().Header().Set("Server-Timing",
					fmt.Sprintf("total;dur=%d;desc=\"Total\"", dur))
			})
			return next(c)
		}
	}
}
