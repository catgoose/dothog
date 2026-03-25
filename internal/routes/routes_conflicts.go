// setup:feature:sync
package routes

import (
	"net/http"
	"strconv"

	"catgoose/dothog/internal/logger"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/hypermedia"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

// Conflict represents an unresolved sync conflict stored in memory.
// In a production app, these would be persisted to a database table.
type Conflict struct {
	ID            int    `json:"id"`
	Method        string `json:"method"`
	URL           string `json:"url"`
	ClientData    string `json:"client_data"`
	ServerData    string `json:"server_data"`
	ClientVersion int    `json:"client_version"`
	ServerVersion int    `json:"server_version"`
	QueuedAt      string `json:"queued_at"`
	DetectedAt    string `json:"detected_at"`
	Status        string `json:"status"` // pending, resolved_mine, resolved_theirs
}

func (ar *appRoutes) initConflictRoutes() {
	ar.e.GET("/conflicts", ar.handleConflictsList)
	ar.e.GET("/conflicts/:id", ar.handleConflictDetail)
	ar.e.POST("/conflicts/:id/resolve", ar.handleConflictResolve)
}

func (ar *appRoutes) handleConflictsList(c echo.Context) error {
	// TODO: fetch conflicts from database
	// For now, render an empty list page
	return handler.RenderComponent(c, views.ConflictsPage(nil))
}

func (ar *appRoutes) handleConflictDetail(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusBadRequest, "Invalid conflict ID", err,
			hypermedia.BackButton("Go back"),
		)
	}

	_ = id // TODO: look up conflict by ID
	log := logger.WithContext(c.Request().Context())
	log.Info("Viewing conflict", "id", id)

	// Placeholder: return not found until persistence is implemented
	return handler.HandleHypermediaError(c, http.StatusNotFound, "Conflict not found", nil,
		hypermedia.BackButton("Go back"),
		hypermedia.GoHomeButton("Conflicts", "/conflicts", "#main"),
	)
}

func (ar *appRoutes) handleConflictResolve(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusBadRequest, "Invalid conflict ID", err)
	}

	resolution := c.FormValue("resolution") // "mine" or "theirs"
	log := logger.WithContext(c.Request().Context())
	log.Info("Resolving conflict", "id", id, "resolution", resolution)

	// TODO: apply resolution, update conflict status, push via SSE
	_ = id
	_ = resolution

	// Redirect to conflicts list after resolution
	return c.Redirect(http.StatusSeeOther, "/conflicts")
}
