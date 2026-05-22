// setup:feature:avatar

package routes

import (
	"net/http"

	"catgoose/dothog/internal/service/graph"

	"github.com/labstack/echo/v4"
)

// RegisterAvatarRoutes wires GET /api/avatar/:azureID; the handler is closed
// over store so callers don't need to re-pass it per request.
func RegisterAvatarRoutes(e *echo.Echo, store *graph.PhotoStore) {
	e.GET("/api/avatar/:azureID", handleAvatar(store))
}

func handleAvatar(store *graph.PhotoStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		azureID := c.Param("azureID")
		if azureID == "" {
			return c.NoContent(http.StatusBadRequest)
		}
		if !store.HasPhoto(azureID) {
			return c.NoContent(http.StatusNotFound)
		}
		return c.File(store.PhotoPath(azureID))
	}
}
