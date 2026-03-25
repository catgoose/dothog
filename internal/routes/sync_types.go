// setup:feature:sync
package routes

import "time"

// SyncOperation represents a single offline mutation queued by the client.
type SyncOperation struct {
	Method      string `json:"method"`       // POST, PUT, DELETE
	URL         string `json:"url"`          // e.g. /demo/repository/tasks/42
	Body        string `json:"body"`         // form-urlencoded payload
	ContentType string `json:"content_type"` // Content-Type header
	Version     *int   `json:"version"`      // row version at time of edit (nil for creates)
	QueuedAt    string `json:"queued_at"`    // ISO 8601 timestamp from client
}

// SyncRequest is the batch payload sent by the client's sync manager.
type SyncRequest struct {
	Operations    []SyncOperation `json:"operations"`
	SchemaVersion int             `json:"schema_version"`
}

// SyncResultStatus represents the outcome of a sync operation.
type SyncResultStatus string

const (
	SyncApplied  SyncResultStatus = "applied"
	SyncConflict SyncResultStatus = "conflict"
	SyncRejected SyncResultStatus = "rejected"
	SyncError    SyncResultStatus = "error"
)

// SyncResult is the server's response for a single operation.
type SyncResult struct {
	Index      int              `json:"index"`
	Status     SyncResultStatus `json:"status"`
	Message    string           `json:"message,omitempty"`
	NewVersion int              `json:"new_version,omitempty"`
}

// SyncResponse is the batch response sent back to the client.
type SyncResponse struct {
	Results   []SyncResult `json:"results"`
	Timestamp time.Time    `json:"server_time"`
}
