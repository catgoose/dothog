package schema

import (
	"testing"

	"catgoose/go-htmx-demo/internals/database/dialect"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLookupTable(t *testing.T) {
	td := NewLookupTable("Tags", "Type", "Label")

	cols := td.SelectColumns()
	assert.Equal(t, []string{"ID", "Type", "Label"}, cols)

	// ID excluded from insert
	insert := td.InsertColumns()
	assert.Equal(t, []string{"Type", "Label"}, insert)

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 3) // CREATE TABLE + 2 indexes

	assert.Contains(t, stmts[0], "CREATE TABLE Tags")
	assert.Contains(t, stmts[0], "Type TEXT NOT NULL")
	assert.Contains(t, stmts[0], "Label TEXT NOT NULL")
	assert.Contains(t, stmts[1], "idx_tags_type")
	assert.Contains(t, stmts[2], "idx_tags_type_label")
}

func TestNewLookupTable_CustomColumns(t *testing.T) {
	td := NewLookupTable("Lookups", "Category", "Name")

	cols := td.SelectColumns()
	assert.Equal(t, []string{"ID", "Category", "Name"}, cols)

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 3)

	assert.Contains(t, stmts[0], "Category TEXT NOT NULL")
	assert.Contains(t, stmts[0], "Name TEXT NOT NULL")
	assert.Contains(t, stmts[1], "idx_lookups_category")
	assert.Contains(t, stmts[2], "idx_lookups_category_name")
}

func TestNewLookupTable_MSSQL(t *testing.T) {
	td := NewLookupTable("Lookups", "Type", "Label")

	d := dialect.MSSQLDialect{}
	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 3)

	assert.Contains(t, stmts[0], "Type VARCHAR(100) NOT NULL")
	assert.Contains(t, stmts[0], "Label VARCHAR(255) NOT NULL")
}

func TestNewLookupJoinTable(t *testing.T) {
	td := NewLookupJoinTable("ItemTags")

	cols := td.SelectColumns()
	assert.Equal(t, []string{"OwnerID", "LookupID"}, cols)

	// Both columns are immutable
	assert.Empty(t, td.UpdateColumns())

	// Both included in insert (no auto-increment)
	assert.Equal(t, []string{"OwnerID", "LookupID"}, td.InsertColumns())

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 3) // CREATE TABLE + 2 indexes

	assert.Contains(t, stmts[0], "OwnerID INTEGER NOT NULL")
	assert.Contains(t, stmts[0], "LookupID INTEGER NOT NULL")
	assert.Contains(t, stmts[1], "idx_itemtags_ownerid")
	assert.Contains(t, stmts[2], "idx_itemtags_lookupid")
}
