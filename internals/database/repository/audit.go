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
