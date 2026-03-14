package schema

import (
	"testing"

	"catgoose/dothog/internal/database/dialect"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQueueTable(t *testing.T) {
	td := NewQueueTable("JobQueue", "Payload")

	cols := td.SelectColumns()
	assert.Equal(t, []string{"ID", "Payload", "Status", "RetryCount", "ScheduledAt", "ProcessedAt", "CreatedAt"}, cols)

	// ID and CreatedAt excluded from insert
	insert := td.InsertColumns()
	assert.Equal(t, []string{"Payload", "Status", "RetryCount", "ScheduledAt", "ProcessedAt", "CreatedAt"}, insert)

	// CreatedAt is immutable, ID is auto-incr
	update := td.UpdateColumns()
	assert.Contains(t, update, "Payload")
	assert.Contains(t, update, "Status")
	assert.Contains(t, update, "RetryCount")
	assert.Contains(t, update, "ScheduledAt")
	assert.Contains(t, update, "ProcessedAt")
	assert.NotContains(t, update, "CreatedAt")
	assert.NotContains(t, update, "ID")

	d := dialect.SQLiteDialect{}
	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 4) // CREATE TABLE + 3 indexes

	assert.Contains(t, stmts[0], "CREATE TABLE JobQueue")
	assert.Contains(t, stmts[0], "Payload TEXT NOT NULL")
	assert.Contains(t, stmts[0], "Status TEXT NOT NULL DEFAULT 'pending'")
	assert.Contains(t, stmts[0], "RetryCount INTEGER NOT NULL DEFAULT 0")
	assert.Contains(t, stmts[0], "ScheduledAt TIMESTAMP")
	assert.Contains(t, stmts[0], "ProcessedAt TIMESTAMP")
	assert.Contains(t, stmts[0], "CreatedAt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP")
	assert.Contains(t, stmts[1], "idx_jobqueue_status")
	assert.Contains(t, stmts[2], "idx_jobqueue_scheduledat")
	assert.Contains(t, stmts[3], "idx_jobqueue_status_scheduledat")
}

func TestNewQueueTable_MSSQL(t *testing.T) {
	td := NewQueueTable("Outbox", "EventData")

	d := dialect.MSSQLDialect{}
	stmts := td.CreateSQL(d)
	require.Len(t, stmts, 4)

	assert.Contains(t, stmts[0], "EventData NVARCHAR(MAX) NOT NULL")
	assert.Contains(t, stmts[0], "Status VARCHAR(50) NOT NULL DEFAULT 'pending'")
	assert.Contains(t, stmts[0], "CreatedAt DATETIME NOT NULL DEFAULT GETDATE()")
}
