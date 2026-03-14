// setup:feature:database
package repository

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWhere_Empty(t *testing.T) {
	w := NewWhere()
	assert.Empty(t, w.String())
	assert.Empty(t, w.Args())
	assert.False(t, w.HasConditions())
}

func TestWhereBuilder_And(t *testing.T) {
	w := NewWhere().
		And("Active = @Active", sql.Named("Active", true)).
		And("RoleID = @RoleID", sql.Named("RoleID", 3))

	assert.Equal(t, "WHERE Active = @Active AND RoleID = @RoleID", w.String())
	assert.Len(t, w.Args(), 2)
	assert.True(t, w.HasConditions())
}

func TestWhereBuilder_AndIf(t *testing.T) {
	w := NewWhere().
		And("1=1").
		AndIf(false, "Skipped = @Skipped", sql.Named("Skipped", true)).
		AndIf(true, "Included = @Included", sql.Named("Included", true))

	assert.Contains(t, w.String(), "Included = @Included")
	assert.NotContains(t, w.String(), "Skipped")
	assert.Len(t, w.Args(), 1)
}

func TestWhereBuilder_Or(t *testing.T) {
	w := NewWhere().
		And("Name = @Name", sql.Named("Name", "Alice")).
		Or("Email = @Email", sql.Named("Email", "alice@example.com"))

	assert.Equal(t, "WHERE Name = @Name OR Email = @Email", w.String())
	assert.Len(t, w.Args(), 2)
}

func TestWhereBuilder_OrIf(t *testing.T) {
	w := NewWhere().
		And("Name = @Name", sql.Named("Name", "Bob")).
		OrIf(false, "Skipped = @Skipped").
		OrIf(true, "Email = @Email", sql.Named("Email", "bob@example.com"))

	assert.Contains(t, w.String(), "Email = @Email")
	assert.NotContains(t, w.String(), "Skipped")
	assert.Len(t, w.Args(), 2)
}

func TestWhereBuilder_Or_AsFirstClause(t *testing.T) {
	w := NewWhere().Or("Standalone = @Standalone", sql.Named("Standalone", 1))
	assert.Equal(t, "WHERE Standalone = @Standalone", w.String())
}

func TestWhereBuilder_Search(t *testing.T) {
	w := NewWhere().
		And("Active = 1").
		Search("foo", "Name", "Email")

	assert.Contains(t, w.String(), "WHERE Active = 1 AND")
	assert.Contains(t, w.String(), "Name LIKE @SearchPattern")
	assert.Contains(t, w.String(), "Email LIKE @SearchPattern")
	assert.Len(t, w.Args(), 2) // Search, SearchPattern
}

func TestWhereBuilder_Search_EmptySearch(t *testing.T) {
	w := NewWhere().Search("", "Name")
	assert.Empty(t, w.String())
	assert.False(t, w.HasConditions())
}

func TestWhereBuilder_Search_NoFields(t *testing.T) {
	w := NewWhere().Search("foo")
	assert.Empty(t, w.String())
	assert.False(t, w.HasConditions())
}

func TestWhereBuilder_NotDeleted(t *testing.T) {
	w := NewWhere().NotDeleted()
	assert.Equal(t, "WHERE DeletedAt IS NULL", w.String())
	assert.Empty(t, w.Args())
}

func TestWhereBuilder_NotDeleted_WithOtherConditions(t *testing.T) {
	w := NewWhere().
		And("Active = @Active", sql.Named("Active", true)).
		NotDeleted()
	assert.Equal(t, "WHERE Active = @Active AND DeletedAt IS NULL", w.String())
	assert.Len(t, w.Args(), 1)
}

func TestWhereBuilder_NotExpired(t *testing.T) {
	w := NewWhere().NotExpired()
	assert.Equal(t, "WHERE (ExpiresAt IS NULL OR ExpiresAt > CURRENT_TIMESTAMP)", w.String())
	assert.Empty(t, w.Args())
}

func TestWhereBuilder_HasStatus(t *testing.T) {
	w := NewWhere().HasStatus("active")
	assert.Equal(t, "WHERE Status = @Status", w.String())
	assert.Len(t, w.Args(), 1)
}

func TestWhereBuilder_IsRoot(t *testing.T) {
	w := NewWhere().IsRoot()
	assert.Equal(t, "WHERE ParentID IS NULL", w.String())
	assert.Empty(t, w.Args())
}

func TestWhereBuilder_HasParent(t *testing.T) {
	w := NewWhere().HasParent(42)
	assert.Equal(t, "WHERE ParentID = @ParentID", w.String())
	assert.Len(t, w.Args(), 1)
}

func TestWhereBuilder_HasVersion(t *testing.T) {
	w := NewWhere().HasVersion(3)
	assert.Equal(t, "WHERE Version = @Version", w.String())
	assert.Len(t, w.Args(), 1)
}

func TestWhereBuilder_CombinedTraits(t *testing.T) {
	w := NewWhere().
		NotDeleted().
		NotExpired().
		HasStatus("active").
		IsRoot()
	assert.Contains(t, w.String(), "DeletedAt IS NULL")
	assert.Contains(t, w.String(), "ExpiresAt IS NULL OR ExpiresAt > CURRENT_TIMESTAMP")
	assert.Contains(t, w.String(), "Status = @Status")
	assert.Contains(t, w.String(), "ParentID IS NULL")
	assert.Len(t, w.Args(), 1) // only Status has an arg
}
