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

// InitVersion sets Version to 1 for a new record.
func InitVersion(version *int) {
	if version != nil {
		*version = 1
	}
}

// IncrementVersion increments Version by 1.
func IncrementVersion(version *int) {
	if version != nil {
		*version++
	}
}

// SetSortOrder sets the SortOrder value.
func SetSortOrder(sortOrder *int, order int) {
	if sortOrder != nil {
		*sortOrder = order
	}
}

// SetStatus sets the Status value.
func SetStatus(status *string, value string) {
	if status != nil {
		*status = value
	}
}

// SetExpiry sets ExpiresAt to the given time.
func SetExpiry(expiresAt *time.Time, t time.Time) {
	if expiresAt != nil {
		*expiresAt = t
	}
}

// SetReplacement sets ReplacedByID to the given ID.
func SetReplacement(replacedByID *int64, id int64) {
	if replacedByID != nil {
		*replacedByID = id
	}
}

// ClearReplacement sets ReplacedByID to zero value.
func ClearReplacement(replacedByID *int64) {
	if replacedByID != nil {
		*replacedByID = 0
	}
}

// SetArchive sets ArchivedAt to the current time.
func SetArchive(archivedAt *time.Time) {
	if archivedAt != nil {
		*archivedAt = GetNow()
	}
}

// ClearArchive sets ArchivedAt to zero value (unarchives).
func ClearArchive(archivedAt *time.Time) {
	if archivedAt != nil {
		*archivedAt = time.Time{}
	}
}

// ClearExpiry sets ExpiresAt to zero value (removes expiry).
func ClearExpiry(expiresAt *time.Time) {
	if expiresAt != nil {
		*expiresAt = time.Time{}
	}
}
