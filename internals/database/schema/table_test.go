package schema

import (
	"strings"
	"testing"

	"catgoose/go-htmx-demo/internals/database/dialect"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableDef_SelectColumns(t *testing.T) {
	td := NewTable("Test").Columns(
		AutoIncrCol("ID"),
		Col("Name", TypeString(255)),
		Col("Email", TypeString(255)),
	)
	assert.Equal(t, []string{"ID", "Name", "Email"}, td.SelectColumns())
}

func TestTableDef_InsertColumns_ExcludesAutoIncr(t *testing.T) {
	td := NewTable("Test").Columns(
		AutoIncrCol("ID"),
		Col("Name", TypeString(255)),
		Col("Email", TypeString(255)),
	)
	cols := td.InsertColumns()
	assert.Equal(t, []string{"Name", "Email"}, cols)
}

func TestTableDef_UpdateColumns_OnlyMutable(t *testing.T) {
	td := NewTable("Test").Columns(
		AutoIncrCol("ID"),
		Col("Name", TypeString(255)),
		Col("Email", TypeString(255)).Immutable(),
	)
	cols := td.UpdateColumns()
	assert.Equal(t, []string{"Name"}, cols)
}

func TestTableDef_WithTimestamps(t *testing.T) {
	td := NewTable("Test").
		Columns(AutoIncrCol("ID")).
		WithTimestamps()

	cols := td.SelectColumns()
	assert.Contains(t, cols, "CreatedAt")
	assert.Contains(t, cols, "UpdatedAt")

	// CreatedAt should be immutable, UpdatedAt should be mutable
	update := td.UpdateColumns()
	assert.NotContains(t, update, "CreatedAt")
	assert.Contains(t, update, "UpdatedAt")
}

func TestTableDef_WithSoftDelete(t *testing.T) {
	td := NewTable("Test").
		Columns(AutoIncrCol("ID")).
		WithSoftDelete()

	assert.True(t, td.HasSoftDelete())
	assert.Contains(t, td.SelectColumns(), "DeletedAt")
}

func TestTableDef_WithAuditTrail(t *testing.T) {
	td := NewTable("Test").
		Columns(AutoIncrCol("ID")).
		WithAuditTrail()

	cols := td.SelectColumns()
	assert.Contains(t, cols, "CreatedBy")
	assert.Contains(t, cols, "UpdatedBy")
	assert.Contains(t, cols, "DeletedBy")

	// CreatedBy immutable, UpdatedBy and DeletedBy mutable
	update := td.UpdateColumns()
	assert.NotContains(t, update, "CreatedBy")
	assert.Contains(t, update, "UpdatedBy")
	assert.Contains(t, update, "DeletedBy")
}

func TestTableDef_CreateSQL_SQLite(t *testing.T) {
	d := dialect.SQLiteDialect{}
	td := NewTable("Items").
		Columns(
			AutoIncrCol("ID"),
			Col("Name", TypeString(255)).NotNull(),
		).
		WithTimestamps().
		Indexes(Index("idx_items_name", "Name"))

	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 2)

	// CREATE TABLE
	assert.Contains(t, stmts[0], "CREATE TABLE Items")
	assert.Contains(t, stmts[0], "ID INTEGER PRIMARY KEY AUTOINCREMENT")
	assert.Contains(t, stmts[0], "Name TEXT NOT NULL")
	assert.Contains(t, stmts[0], "CreatedAt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP")
	assert.Contains(t, stmts[0], "UpdatedAt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP")

	// CREATE INDEX
	assert.Contains(t, stmts[1], "CREATE INDEX IF NOT EXISTS idx_items_name ON Items(Name)")
}

func TestTableDef_CreateSQL_MSSQL(t *testing.T) {
	d := dialect.MSSQLDialect{}
	td := NewTable("Items").
		Columns(
			AutoIncrCol("ID"),
			Col("Name", TypeString(255)).NotNull(),
		).
		WithTimestamps()

	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 1)

	assert.Contains(t, stmts[0], "ID INT PRIMARY KEY IDENTITY(1,1)")
	assert.Contains(t, stmts[0], "Name NVARCHAR(255) NOT NULL")
	assert.Contains(t, stmts[0], "CreatedAt DATETIME NOT NULL DEFAULT GETDATE()")
}

func TestTableDef_DropSQL(t *testing.T) {
	d := dialect.SQLiteDialect{}
	td := NewTable("Items")
	assert.Equal(t, "DROP TABLE IF EXISTS Items", td.DropSQL(d))
}

func TestUsersTable_CreateSQL_MatchesExisting(t *testing.T) {
	d := dialect.SQLiteDialect{}
	stmts := UsersTable.CreateSQL(d)

	// Should have 1 CREATE TABLE + 5 CREATE INDEX statements
	require.Len(t, stmts, 6)

	createSQL := stmts[0]

	// Verify all columns are present
	expectedCols := []string{
		"ID INTEGER PRIMARY KEY AUTOINCREMENT",
		"AzureId TEXT NOT NULL UNIQUE",
		"GivenName TEXT",
		"Surname TEXT",
		"DisplayName TEXT",
		"UserPrincipalName TEXT NOT NULL",
		"Mail TEXT",
		"JobTitle TEXT",
		"OfficeLocation TEXT",
		"Department TEXT",
		"CompanyName TEXT",
		"AccountName TEXT",
		"LastLoginAt TIMESTAMP",
		"CreatedAt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
		"UpdatedAt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
	}
	for _, col := range expectedCols {
		assert.Contains(t, createSQL, col, "missing column: %s", col)
	}

	// Verify indexes
	idxStmts := strings.Join(stmts[1:], "\n")
	assert.Contains(t, idxStmts, "idx_users_azureid")
	assert.Contains(t, idxStmts, "idx_users_userprincipalname")
	assert.Contains(t, idxStmts, "idx_users_displayname")
	assert.Contains(t, idxStmts, "idx_users_mail")
	assert.Contains(t, idxStmts, "idx_users_lastloginat")
}

func TestUsersTable_InsertColumns(t *testing.T) {
	cols := UsersTable.InsertColumns()
	assert.NotContains(t, cols, "ID")
	assert.Contains(t, cols, "AzureId")
	assert.Contains(t, cols, "CreatedAt")
	assert.Contains(t, cols, "UpdatedAt")
}

func TestUsersTable_UpdateColumns(t *testing.T) {
	cols := UsersTable.UpdateColumns()
	assert.NotContains(t, cols, "ID")
	assert.NotContains(t, cols, "CreatedAt")
	assert.Contains(t, cols, "AzureId")
	assert.Contains(t, cols, "UpdatedAt")
}

func TestTableDef_TraitsComposition(t *testing.T) {
	td := NewTable("FullFeatured").
		Columns(
			AutoIncrCol("ID"),
			Col("Name", TypeString(255)).NotNull(),
		).
		WithTimestamps().
		WithSoftDelete().
		WithAuditTrail()

	all := td.SelectColumns()
	assert.Equal(t, []string{
		"ID", "Name",
		"CreatedAt", "UpdatedAt",
		"DeletedAt",
		"CreatedBy", "UpdatedBy", "DeletedBy",
	}, all)

	insert := td.InsertColumns()
	assert.NotContains(t, insert, "ID")
	assert.Len(t, insert, 7) // Name, CreatedAt, UpdatedAt, DeletedAt, CreatedBy, UpdatedBy, DeletedBy

	update := td.UpdateColumns()
	assert.NotContains(t, update, "ID")
	assert.NotContains(t, update, "CreatedAt")
	assert.NotContains(t, update, "CreatedBy")
	assert.Contains(t, update, "Name")
	assert.Contains(t, update, "UpdatedAt")
	assert.Contains(t, update, "DeletedAt")
	assert.Contains(t, update, "UpdatedBy")
	assert.Contains(t, update, "DeletedBy")

	assert.True(t, td.HasSoftDelete())
}
