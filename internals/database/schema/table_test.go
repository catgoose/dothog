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

func TestTableDef_WithVersion(t *testing.T) {
	td := NewTable("Test").
		Columns(AutoIncrCol("ID")).
		WithVersion()

	assert.True(t, td.HasVersion())
	assert.Contains(t, td.SelectColumns(), "Version")
	assert.Contains(t, td.UpdateColumns(), "Version")

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	assert.Contains(t, stmts[0], "Version INTEGER NOT NULL DEFAULT 1")
}

func TestTableDef_WithSortOrder(t *testing.T) {
	td := NewTable("Test").
		Columns(AutoIncrCol("ID")).
		WithSortOrder()

	assert.Contains(t, td.SelectColumns(), "SortOrder")
	assert.Contains(t, td.UpdateColumns(), "SortOrder")

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	assert.Contains(t, stmts[0], "SortOrder INTEGER NOT NULL DEFAULT 0")
}

func TestTableDef_WithStatus(t *testing.T) {
	td := NewTable("Test").
		Columns(AutoIncrCol("ID")).
		WithStatus("draft")

	assert.Contains(t, td.SelectColumns(), "Status")
	assert.Contains(t, td.UpdateColumns(), "Status")

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	assert.Contains(t, stmts[0], "Status TEXT NOT NULL DEFAULT 'draft'")
}

func TestTableDef_WithNotes(t *testing.T) {
	td := NewTable("Test").
		Columns(AutoIncrCol("ID")).
		WithNotes()

	assert.Contains(t, td.SelectColumns(), "Notes")
	assert.Contains(t, td.UpdateColumns(), "Notes")

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	assert.Contains(t, stmts[0], "Notes TEXT")
}

func TestTableDef_WithUUID(t *testing.T) {
	td := NewTable("Test").
		Columns(AutoIncrCol("ID")).
		WithUUID()

	assert.Contains(t, td.SelectColumns(), "UUID")
	// UUID is immutable
	assert.NotContains(t, td.UpdateColumns(), "UUID")
	// UUID is included in insert
	assert.Contains(t, td.InsertColumns(), "UUID")

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	assert.Contains(t, stmts[0], "UUID TEXT NOT NULL UNIQUE")
}

func TestTableDef_WithParent(t *testing.T) {
	td := NewTable("Test").
		Columns(AutoIncrCol("ID")).
		WithParent()

	assert.Contains(t, td.SelectColumns(), "ParentID")
	assert.Contains(t, td.UpdateColumns(), "ParentID")

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	assert.Contains(t, stmts[0], "ParentID INTEGER")
}

func TestTableDef_WithExpiry(t *testing.T) {
	td := NewTable("Test").
		Columns(AutoIncrCol("ID")).
		WithExpiry()

	assert.True(t, td.HasExpiry())
	assert.Contains(t, td.SelectColumns(), "ExpiresAt")
	assert.Contains(t, td.UpdateColumns(), "ExpiresAt")

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	assert.Contains(t, stmts[0], "ExpiresAt TIMESTAMP")
}

func TestTableDef_WithArchive(t *testing.T) {
	td := NewTable("Test").
		Columns(AutoIncrCol("ID")).
		WithArchive()

	assert.True(t, td.HasArchive())
	assert.Contains(t, td.SelectColumns(), "ArchivedAt")
	assert.Contains(t, td.UpdateColumns(), "ArchivedAt")

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	assert.Contains(t, stmts[0], "ArchivedAt TIMESTAMP")
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

func TestTableDef_UniqueColumns(t *testing.T) {
	d := dialect.SQLiteDialect{}
	td := NewTable("JoinTable").
		Columns(
			Col("OwnerID", TypeInt()).NotNull(),
			Col("TagID", TypeInt()).NotNull(),
		).
		UniqueColumns("OwnerID", "TagID")

	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 1)
	assert.Contains(t, stmts[0], "UNIQUE (OwnerID, TagID)")
}

func TestTableDef_References_SQLite(t *testing.T) {
	d := dialect.SQLiteDialect{}
	td := NewTable("Orders").
		Columns(
			AutoIncrCol("ID"),
			Col("UserID", TypeInt()).NotNull().References("Users", "ID"),
		)

	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 1)
	assert.Contains(t, stmts[0], "UserID INTEGER NOT NULL REFERENCES Users(ID)")
}

func TestTableDef_References_MSSQL(t *testing.T) {
	d := dialect.MSSQLDialect{}
	td := NewTable("Orders").
		Columns(
			AutoIncrCol("ID"),
			Col("UserID", TypeInt()).NotNull().References("Users", "ID"),
		)

	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 1)
	assert.Contains(t, stmts[0], "UserID INT NOT NULL REFERENCES Users(ID)")
}

func TestTableDef_CreateIfNotExistsSQL_SQLite(t *testing.T) {
	d := dialect.SQLiteDialect{}
	td := NewTable("Items").
		Columns(
			AutoIncrCol("ID"),
			Col("Name", TypeString(255)).NotNull(),
		).
		Indexes(Index("idx_items_name", "Name"))

	stmts := td.CreateIfNotExistsSQL(d)
	require.Len(t, stmts, 2)
	assert.Contains(t, stmts[0], "CREATE TABLE IF NOT EXISTS Items")
	assert.Contains(t, stmts[0], "Name TEXT NOT NULL")
	assert.Contains(t, stmts[1], "CREATE INDEX IF NOT EXISTS idx_items_name")
}

func TestTableDef_CreateIfNotExistsSQL_MSSQL(t *testing.T) {
	d := dialect.MSSQLDialect{}
	td := NewTable("Items").
		Columns(
			AutoIncrCol("ID"),
			Col("Name", TypeString(255)).NotNull(),
		)

	stmts := td.CreateIfNotExistsSQL(d)
	require.Len(t, stmts, 1)
	assert.Contains(t, stmts[0], "IF NOT EXISTS")
	assert.Contains(t, stmts[0], "OBJECT_ID(N'[dbo].[Items]')")
	assert.Contains(t, stmts[0], "CREATE TABLE Items")
}

func TestTableDef_WithReplacement(t *testing.T) {
	td := NewTable("Test").
		Columns(AutoIncrCol("ID")).
		WithReplacement()

	assert.Contains(t, td.SelectColumns(), "ReplacedByID")
	assert.Contains(t, td.UpdateColumns(), "ReplacedByID")

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	assert.Contains(t, stmts[0], "ReplacedByID INTEGER")
}

func TestTableDef_WithSeedRows(t *testing.T) {
	td := NewConfigTable("Settings", "Key", "Value").
		WithSeedRows(
			SeedRow{"Key": "'app.name'", "Value": "'My App'"},
			SeedRow{"Key": "'app.version'", "Value": "'1.0.0'"},
		)

	assert.True(t, td.HasSeedData())
	assert.Len(t, td.SeedRows(), 2)

	stmts := td.SeedSQL()
	require.Len(t, stmts, 2)
	assert.Contains(t, stmts[0], "INSERT OR IGNORE INTO Settings")
	assert.Contains(t, stmts[0], "'app.name'")
	assert.Contains(t, stmts[0], "'My App'")
	assert.Contains(t, stmts[1], "'app.version'")
	assert.Contains(t, stmts[1], "'1.0.0'")
}

func TestTableDef_WithSeedRows_NullDefaults(t *testing.T) {
	td := NewLookupTable("Tags", "Type", "Label").
		WithSeedRows(
			SeedRow{"Type": "'color'", "Label": "'Red'"},
			SeedRow{"Type": "'color'"}, // Label not specified -> NULL
		)

	stmts := td.SeedSQL()
	require.Len(t, stmts, 2)
	assert.Contains(t, stmts[0], "'Red'")
	assert.Contains(t, stmts[1], "NULL")
}

func TestTableDef_NoSeedData(t *testing.T) {
	td := NewTable("Test").Columns(AutoIncrCol("ID"))

	assert.False(t, td.HasSeedData())
	assert.Nil(t, td.SeedSQL())
}

func TestTableDef_ArchiveWithReplacement(t *testing.T) {
	td := NewTable("Versioned").
		Columns(
			AutoIncrCol("ID"),
			Col("Name", TypeString(255)).NotNull(),
		).
		WithTimestamps().
		WithArchive().
		WithReplacement()

	assert.True(t, td.HasArchive())
	assert.Contains(t, td.SelectColumns(), "ArchivedAt")
	assert.Contains(t, td.SelectColumns(), "ReplacedByID")

	// Both should be mutable
	update := td.UpdateColumns()
	assert.Contains(t, update, "ArchivedAt")
	assert.Contains(t, update, "ReplacedByID")

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	assert.Contains(t, stmts[0], "ArchivedAt TIMESTAMP")
	assert.Contains(t, stmts[0], "ReplacedByID INTEGER")
}
