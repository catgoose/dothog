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
