// setup:feature:graph

package domain

import (
	"database/sql"
	"time"
)

// Timestamps provides CreatedAt and UpdatedAt fields for embedding in domain models.
type Timestamps struct {
	CreatedAt time.Time `db:"CreatedAt" json:"created_at"`
	UpdatedAt time.Time `db:"UpdatedAt" json:"updated_at"`
}

// SoftDelete provides a nullable DeletedAt field for embedding in domain models.
type SoftDelete struct {
	DeletedAt sql.NullTime `db:"DeletedAt" json:"deleted_at,omitzero"`
}

// AuditTrail provides CreatedBy, UpdatedBy, and DeletedBy fields for embedding in domain models.
type AuditTrail struct {
	CreatedBy sql.NullString `db:"CreatedBy" json:"created_by,omitzero"`
	UpdatedBy sql.NullString `db:"UpdatedBy" json:"updated_by,omitzero"`
	DeletedBy sql.NullString `db:"DeletedBy" json:"deleted_by,omitzero"`
}

// Version provides an optimistic concurrency control field for embedding in domain models.
type Version struct {
	Version int `db:"Version" json:"version"`
}

// SortOrder provides a manual ordering field for embedding in domain models.
type SortOrder struct {
	SortOrder int `db:"SortOrder" json:"sort_order"`
}

// Status provides a status field for embedding in domain models.
type Status struct {
	Status string `db:"Status" json:"status"`
}

// Notes provides a nullable notes field for embedding in domain models.
type Notes struct {
	Notes sql.NullString `db:"Notes" json:"notes,omitzero"`
}

// UUID provides a unique identifier field for embedding in domain models.
type UUID struct {
	UUID string `db:"UUID" json:"uuid"`
}

// Parent provides a nullable parent reference for tree structures.
type Parent struct {
	ParentID sql.NullInt64 `db:"ParentID" json:"parent_id,omitzero"`
}

// Expiry provides a nullable expiration timestamp for embedding in domain models.
type Expiry struct {
	ExpiresAt sql.NullTime `db:"ExpiresAt" json:"expires_at,omitzero"`
}

// ToNullString converts a string to sql.NullString. Empty strings are treated as null.
func ToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
