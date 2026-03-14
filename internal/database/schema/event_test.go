package schema

import (
	"testing"

	"catgoose/dothog/internal/database/dialect"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventTable(t *testing.T) {
	td := NewEventTable("AuditLog",
		Col("EventType", TypeVarchar(100)).NotNull(),
		Col("Actor", TypeVarchar(255)),
		Col("Payload", TypeText()),
	)

	cols := td.SelectColumns()
	assert.Equal(t, []string{"ID", "EventType", "Actor", "Payload", "CreatedAt"}, cols)

	// All columns are immutable — no update columns
	assert.Empty(t, td.UpdateColumns())

	// ID excluded from insert, all others included
	insert := td.InsertColumns()
	assert.Equal(t, []string{"EventType", "Actor", "Payload", "CreatedAt"}, insert)

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 2) // CREATE TABLE + 1 index

	assert.Contains(t, stmts[0], "CREATE TABLE AuditLog")
	assert.Contains(t, stmts[0], "EventType TEXT NOT NULL")
	assert.Contains(t, stmts[0], "Payload TEXT")
	assert.Contains(t, stmts[0], "CreatedAt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP")
	assert.Contains(t, stmts[1], "idx_auditlog_createdat")
}

func TestNewEventTable_MSSQL(t *testing.T) {
	td := NewEventTable("Events",
		Col("Type", TypeVarchar(50)).NotNull(),
		Col("Data", TypeText()),
	)

	d := dialect.MSSQLDialect{}
	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 2)

	assert.Contains(t, stmts[0], "Type VARCHAR(50) NOT NULL")
	assert.Contains(t, stmts[0], "CreatedAt DATETIME NOT NULL DEFAULT GETDATE()")
}
