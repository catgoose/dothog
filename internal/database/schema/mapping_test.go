package schema

import (
	"testing"

	"catgoose/dothog/internal/database/dialect"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMappingTable(t *testing.T) {
	td := NewMappingTable("UserRoles", "UserID", "RoleID")

	cols := td.SelectColumns()
	assert.Equal(t, []string{"UserID", "RoleID"}, cols)

	// Both columns are immutable
	assert.Empty(t, td.UpdateColumns())

	// Both included in insert (no auto-increment)
	assert.Equal(t, []string{"UserID", "RoleID"}, td.InsertColumns())

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 3) // CREATE TABLE + 2 indexes

	assert.Contains(t, stmts[0], "CREATE TABLE UserRoles")
	assert.Contains(t, stmts[0], "UserID INTEGER NOT NULL")
	assert.Contains(t, stmts[0], "RoleID INTEGER NOT NULL")
	assert.Contains(t, stmts[0], "UNIQUE (UserID, RoleID)")
	assert.Contains(t, stmts[1], "idx_userroles_userid")
	assert.Contains(t, stmts[2], "idx_userroles_roleid")
}

func TestNewMappingTable_MSSQL(t *testing.T) {
	td := NewMappingTable("ProjectMembers", "ProjectID", "MemberID")

	d := dialect.MSSQLDialect{}
	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 3)

	assert.Contains(t, stmts[0], "ProjectID INT NOT NULL")
	assert.Contains(t, stmts[0], "MemberID INT NOT NULL")
	assert.Contains(t, stmts[0], "UNIQUE (ProjectID, MemberID)")
}
