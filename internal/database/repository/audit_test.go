package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetCreateTimestamps(t *testing.T) {
	var created, updated time.Time
	SetCreateTimestamps(&created, &updated)
	assert.False(t, created.IsZero())
	assert.False(t, updated.IsZero())
	assert.Equal(t, created, updated)
}

func TestSetUpdateTimestamp(t *testing.T) {
	var updated time.Time
	SetUpdateTimestamp(&updated)
	assert.False(t, updated.IsZero())
}

func TestSetSoftDelete(t *testing.T) {
	var deletedAt time.Time
	SetSoftDelete(&deletedAt)
	assert.False(t, deletedAt.IsZero())
}

func TestSetSoftDelete_Nil(t *testing.T) {
	SetSoftDelete(nil) // should not panic
}

func TestSetDeleteAudit(t *testing.T) {
	var deletedAt time.Time
	var deletedBy string
	SetDeleteAudit(&deletedAt, &deletedBy, "admin")
	assert.False(t, deletedAt.IsZero())
	assert.Equal(t, "admin", deletedBy)
}

func TestSetCreateAudit(t *testing.T) {
	var createdBy, updatedBy string
	SetCreateAudit(&createdBy, &updatedBy, "user1")
	assert.Equal(t, "user1", createdBy)
	assert.Equal(t, "user1", updatedBy)
}

func TestSetUpdateAudit(t *testing.T) {
	var updatedBy string
	SetUpdateAudit(&updatedBy, "user2")
	assert.Equal(t, "user2", updatedBy)
}

func TestSetUpdateAudit_Nil(t *testing.T) {
	SetUpdateAudit(nil, "user2") // should not panic
}

func TestInitVersion(t *testing.T) {
	var v int
	InitVersion(&v)
	assert.Equal(t, 1, v)
}

func TestInitVersion_Nil(t *testing.T) {
	InitVersion(nil) // should not panic
}

func TestIncrementVersion(t *testing.T) {
	v := 3
	IncrementVersion(&v)
	assert.Equal(t, 4, v)
}

func TestIncrementVersion_Nil(t *testing.T) {
	IncrementVersion(nil) // should not panic
}

func TestSetSortOrder(t *testing.T) {
	var s int
	SetSortOrder(&s, 5)
	assert.Equal(t, 5, s)
}

func TestSetStatus(t *testing.T) {
	var s string
	SetStatus(&s, "active")
	assert.Equal(t, "active", s)
}

func TestSetExpiry(t *testing.T) {
	var e time.Time
	future := time.Now().Add(24 * time.Hour)
	SetExpiry(&e, future)
	assert.Equal(t, future, e)
}

func TestClearExpiry(t *testing.T) {
	e := time.Now()
	ClearExpiry(&e)
	assert.True(t, e.IsZero())
}

func TestClearExpiry_Nil(t *testing.T) {
	ClearExpiry(nil) // should not panic
}
