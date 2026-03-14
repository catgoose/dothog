package schema

import (
	"testing"

	"catgoose/dothog/internal/database/dialect"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigTable(t *testing.T) {
	td := NewConfigTable("Settings", "Key", "Value")

	cols := td.SelectColumns()
	assert.Equal(t, []string{"ID", "Key", "Value"}, cols)

	// ID excluded from insert
	insert := td.InsertColumns()
	assert.Equal(t, []string{"Key", "Value"}, insert)

	// Key and Value are mutable
	update := td.UpdateColumns()
	assert.Equal(t, []string{"Key", "Value"}, update)

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 2) // CREATE TABLE + 1 index

	assert.Contains(t, stmts[0], "CREATE TABLE Settings")
	assert.Contains(t, stmts[0], "Key TEXT NOT NULL UNIQUE")
	assert.Contains(t, stmts[0], "Value TEXT")
	assert.Contains(t, stmts[1], "idx_settings_key")
}

func TestNewConfigTable_CustomColumns(t *testing.T) {
	td := NewConfigTable("AppConfig", "Name", "Data")

	cols := td.SelectColumns()
	assert.Equal(t, []string{"ID", "Name", "Data"}, cols)

	d := dialect.MSSQLDialect{}
	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 2)

	assert.Contains(t, stmts[0], "Name VARCHAR(255) NOT NULL UNIQUE")
	assert.Contains(t, stmts[0], "Data NVARCHAR(MAX)")
}
