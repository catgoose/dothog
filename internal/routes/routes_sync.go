// setup:feature:sync
package routes

import (
	"fmt"
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
// It checks row versions against the database to detect conflicts:
//   - Creates (no version) are accepted unconditionally
//   - Unknown resource URLs are accepted without version check
//   - Known resources are checked against the current database version
//   - Version match → applied; mismatch → conflict; row gone → rejected
func (ar *appRoutes) processSyncOperation(c echo.Context, index int, op SyncOperation) SyncResult {
	// Creates (no version) are accepted without version check
	if op.Version == nil {
		return SyncResult{
			Index:   index,
			Status:  SyncApplied,
			Message: "created",
		}
	}

	// Try to parse the URL for version checking
	table, id, ok := parseResourceURL(op.URL)
	if !ok {
		// Unknown resource — accept without version check
		return SyncResult{
			Index:   index,
			Status:  SyncApplied,
			Message: "accepted (unknown resource type)",
		}
	}

	// If no version checker is configured, accept all
	if ar.versionChecker == nil {
		return SyncResult{
			Index:   index,
			Status:  SyncApplied,
			Message: "accepted (no version checker)",
		}
	}

	// Check the current version
	currentVersion, err := ar.versionChecker.CurrentVersion(c.Request().Context(), table, id)
	if err != nil {
		return SyncResult{
			Index:   index,
			Status:  SyncError,
			Message: fmt.Sprintf("version check failed: %v", err),
		}
	}

	// Row not found (deleted)
	if currentVersion == -1 {
		return SyncResult{
			Index:   index,
			Status:  SyncRejected,
			Message: "resource no longer exists",
		}
	}

	// Version mismatch — conflict
	if *op.Version != currentVersion {
		return SyncResult{
			Index:      index,
			Status:     SyncConflict,
			Message:    fmt.Sprintf("version mismatch: client=%d, server=%d", *op.Version, currentVersion),
			NewVersion: currentVersion,
		}
	}

	// Version matches — accept
	return SyncResult{
		Index:      index,
		Status:     SyncApplied,
		Message:    "applied",
		NewVersion: currentVersion + 1,
	}
}
