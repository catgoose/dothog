// setup:feature:avatar

package routes

import (
	"net/http"

	"catgoose/dothog/internal/service/graph"

	"github.com/labstack/echo/v4"
)

// RegisterAvatarRoutes adds the avatar photo endpoint to the Echo instance.
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
