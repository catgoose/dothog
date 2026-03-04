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
