package responsepolicy

import "github.com/labstack/echo/v4"

// Vary appends a single field to the Vary header on every response. Using
// Add (not Set) preserves any prior Vary values upstream middleware or
// handlers already declared, so the cache key shape is the union of all
// declarations rather than a last-write-wins clobber.
func Vary(field string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Add("Vary", field)
			return next(c)
		}
	}
}
