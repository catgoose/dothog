// setup:feature:database

package repository

import (
	"time"
)

// GetNow returns the current time - helper for consistency across repositories
func GetNow() time.Time {
	return time.Now()
}

// SetCreateTimestamps sets CreatedAt and UpdatedAt to current time
// This is a helper that repositories can use in their Create methods
func SetCreateTimestamps(createdAt, updatedAt *time.Time) {
	now := GetNow()
	if createdAt != nil {
		*createdAt = now
	}
	if updatedAt != nil {
		*updatedAt = now
	}
}

// SetUpdateTimestamp sets UpdatedAt to current time
// This is a helper that repositories can use in their Update methods
func SetUpdateTimestamp(updatedAt *time.Time) {
	if updatedAt != nil {
		*updatedAt = GetNow()
	}
}

// SetSoftDelete sets DeletedAt to the current time for soft-delete.
func SetSoftDelete(deletedAt *time.Time) {
	if deletedAt != nil {
		*deletedAt = GetNow()
	}
}

// SetDeleteAudit sets DeletedAt and DeletedBy for a soft-delete with audit trail.
func SetDeleteAudit(deletedAt *time.Time, deletedBy *string, user string) {
	SetSoftDelete(deletedAt)
	if deletedBy != nil {
		*deletedBy = user
	}
}

// SetCreateAudit sets CreatedBy and UpdatedBy for a new record.
func SetCreateAudit(createdBy, updatedBy *string, user string) {
	if createdBy != nil {
		*createdBy = user
	}
	if updatedBy != nil {
		*updatedBy = user
	}
}

// SetUpdateAudit sets UpdatedBy for an updated record.
func SetUpdateAudit(updatedBy *string, user string) {
	if updatedBy != nil {
		*updatedBy = user
	}
}
