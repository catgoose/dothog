// setup:feature:demo

package middleware

import (
	"github.com/catgoose/linkwell"

	"github.com/labstack/echo/v4"
)

// LinkRelationsMiddleware looks up registered link relations for the
// current request path and stores them on the context for template rendering.
// It also emits an RFC 8288 Link HTTP header.
func LinkRelationsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			links := linkwell.LinksFor(path)
			if len(links) > 0 {
				c.Response().Header().Set("Link", linkwell.LinkHeader(links))
				c.Set("link_relations", links)
			}
			return next(c)
		}
	}
}

// GetLinkRelations retrieves link relations from the echo context.
func GetLinkRelations(c echo.Context) []linkwell.LinkRelation {
	if v := c.Get("link_relations"); v != nil {
		if links, ok := v.([]linkwell.LinkRelation); ok {
			return links
		}
	}
	return nil
}
