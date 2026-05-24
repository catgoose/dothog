package responsepolicy

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Preload announces critical assets to the browser before the response body
// starts streaming. On direct HTTP/2+ transports it sends an RFC 8297 103
// Early Hints response so the browser begins fetching while the server is
// still generating; behind the templ proxy (mage watch) 1xx framing gets
// mangled, so it falls back to Link headers on the final response. Either
// way the browser learns about the preloads.
func Preload(links []string, behindProxy bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !behindProxy && c.Request().ProtoMajor >= 2 {
				w := c.Response().Writer
				if flusher, ok := w.(http.Flusher); ok {
					for _, link := range links {
						w.Header().Add("Link", link)
					}
					w.WriteHeader(http.StatusEarlyHints) // 103
					flusher.Flush()
					return next(c)
				}
			}
			for _, link := range links {
				c.Response().Header().Add("Link", link)
			}
			return next(c)
		}
	}
}
