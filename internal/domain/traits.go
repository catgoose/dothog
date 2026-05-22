// setup:feature:graph

package domain

import (
	"database/sql"
	"time"
)

// Timestamps embeds the standard CreatedAt/UpdatedAt audit timestamps.
type Timestamps struct {
	CreatedAt time.Time `db:"CreatedAt" json:"createdAt"`
	UpdatedAt time.Time `db:"UpdatedAt" json:"updatedAt"`
}

// SoftDelete embeds a nullable DeletedAt; non-zero means the row is hidden
// from default queries but recoverable.
type SoftDelete struct {
	DeletedAt sql.NullTime `db:"DeletedAt" json:"deletedAt,omitzero"`
}

// Version embeds an int incremented on every write for optimistic-concurrency checks.
type Version struct {
	Version int `db:"Version" json:"version"`
}

// SortOrder embeds a manual ordering integer; lower values render first.
type SortOrder struct {
	SortOrder int `db:"SortOrder" json:"sortOrder"`
}

// Status embeds a short string state machine column.
type Status struct {
	Status string `db:"Status" json:"status"`
}

// Notes embeds a nullable free-form text field.
type Notes struct {
	Notes sql.NullString `db:"Notes" json:"notes,omitzero"`
}

// Archive is semantically softer than SoftDelete — archived records are hidden
// from default views but remain accessible and restorable.
type Archive struct {
	ArchivedAt sql.NullTime `db:"ArchivedAt" json:"archivedAt,omitzero"`
}

// Replacement embeds an optional pointer to the row that supersedes this one,
// preserving entity lineage when a record is logically replaced.
type Replacement struct {
	ReplacedByID sql.NullInt64 `db:"ReplacedByID" json:"replacedById,omitzero"`
}

// ToNullString treats an empty string as NULL.
func ToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
