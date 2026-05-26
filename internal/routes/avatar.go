// setup:feature:avatar

package routes

import (
	"database/sql"
	"errors"
	"net/http"

	"catgoose/dothog/internal/service/graph"

	"github.com/labstack/echo/v4"
)

// RegisterAvatarRoutes wires GET /api/avatar/:azureID; the handler is closed
// over cache so callers don't need to re-pass it per request. The cache is
// the Graph-owned SQLite PhotoCache — blobs are served directly from the DB.
func RegisterAvatarRoutes(e *echo.Echo, cache *graph.PhotoCache) {
	e.GET("/api/avatar/:azureID", handleAvatar(cache))
}

func handleAvatar(cache *graph.PhotoCache) echo.HandlerFunc {
	return func(c echo.Context) error {
		azureID := c.Param("azureID")
		if azureID == "" {
			return c.NoContent(http.StatusBadRequest)
		}
		data, contentType, err := cache.Get(c.Request().Context(), azureID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.NoContent(http.StatusNotFound)
			}
			return c.NoContent(http.StatusInternalServerError)
		}
		return c.Blob(http.StatusOK, contentType, data)
	}
}
