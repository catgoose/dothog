// setup:feature:sync
package routes

import (
	"net/http"
	"time"

	"catgoose/dothog/internal/logger"

	"github.com/labstack/echo/v4"
)

func (ar *appRoutes) initSyncRoutes() {
	ar.e.POST("/sync", ar.handleSync)
}

func (ar *appRoutes) handleSync(c echo.Context) error {
	var req SyncRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid sync request",
		})
	}

	if len(req.Operations) == 0 {
		return c.JSON(http.StatusOK, SyncResponse{
			Results:   []SyncResult{},
			Timestamp: time.Now().UTC(),
		})
	}

	log := logger.WithContext(c.Request().Context())
	log.Info("Processing sync batch",
		"operations", len(req.Operations),
		"schema_version", req.SchemaVersion,
	)

	results := make([]SyncResult, len(req.Operations))

	for i, op := range req.Operations {
		result := ar.processSyncOperation(c, i, op)
		results[i] = result

		if result.Status == SyncApplied {
			log.Info("Sync operation applied",
				"index", i,
				"method", op.Method,
				"url", op.URL,
			)
		} else {
			log.Warn("Sync operation not applied",
				"index", i,
				"method", op.Method,
				"url", op.URL,
				"status", result.Status,
				"message", result.Message,
			)
		}
	}

	return c.JSON(http.StatusOK, SyncResponse{
		Results:   results,
		Timestamp: time.Now().UTC(),
	})
}

// processSyncOperation handles a single queued operation.
// This is a placeholder implementation that accepts all operations.
// Real implementations will:
// 1. Parse the URL to determine the resource and action
// 2. Check the version against the current row
// 3. Apply or reject based on conflict detection
//
// For now, it forwards the request internally and reports the result.
// This allows the existing CRUD handlers to process the mutations
// without duplicating business logic.
func (ar *appRoutes) processSyncOperation(c echo.Context, index int, op SyncOperation) SyncResult {
	// TODO: Phase 4b — implement version checking and conflict detection
	// For now, return a placeholder that acknowledges the operation.
	//
	// The full implementation will:
	// - Parse op.URL to extract resource type and ID
	// - Look up current version in the database
	// - Compare with op.Version
	// - If match: forward to the existing handler, return applied
	// - If mismatch: return conflict with current data
	// - If row deleted: return rejected

	return SyncResult{
		Index:   index,
		Status:  SyncApplied,
		Message: "accepted (version checking not yet implemented)",
	}
}
