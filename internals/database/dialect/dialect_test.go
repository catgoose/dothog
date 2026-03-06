// setup:feature:database
package dialect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEngine(t *testing.T) {
	tests := []struct {
		input    string
		expected Engine
		wantErr  bool
	}{
		{"sqlserver", MSSQL, false},
		{"mssql", MSSQL, false},
		{"sqlite3", SQLite, false},
		{"sqlite", SQLite, false},
		{"postgres", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			engine, err := ParseEngine(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, engine)
			}
		})
	}
}

func TestNew(t *testing.T) {
	mssql, err := New(MSSQL)
	require.NoError(t, err)
	assert.Equal(t, MSSQL, mssql.Engine())

	sqlite, err := New(SQLite)
	require.NoError(t, err)
	assert.Equal(t, SQLite, sqlite.Engine())

	_, err = New(Engine("unknown"))
	require.Error(t, err)
}

func TestMSSQLDialect(t *testing.T) {
	d := MSSQLDialect{}

	assert.Equal(t, MSSQL, d.Engine())
	assert.Equal(t, "OFFSET @Offset ROWS FETCH NEXT @Limit ROWS ONLY", d.Pagination())
	assert.Equal(t, "INT PRIMARY KEY IDENTITY(1,1)", d.AutoIncrement())
	assert.Equal(t, "GETDATE()", d.Now())
	assert.Equal(t, "DATETIME", d.TimestampType())
	assert.Equal(t, "NVARCHAR(255)", d.StringType(255))
	assert.Equal(t, "VARCHAR(255)", d.VarcharType(255))
	assert.Equal(t, "SELECT SCOPE_IDENTITY() AS ID", d.LastInsertIDQuery())
	assert.False(t, d.SupportsLastInsertID())

	drop := d.DropTableIfExists("Users")
	assert.Contains(t, drop, "OBJECT_ID(N'[dbo].[Users]')")
	assert.Contains(t, drop, "DROP TABLE [dbo].[Users]")

	idx := d.CreateIndexIfNotExists("idx_users_mail", "Users", "Mail")
	assert.Contains(t, idx, "sys.indexes")
	assert.Contains(t, idx, "CREATE INDEX idx_users_mail ON Users(Mail)")
}

func TestSQLiteDialect(t *testing.T) {
	d := SQLiteDialect{}

	assert.Equal(t, SQLite, d.Engine())
	assert.Equal(t, "LIMIT @Limit OFFSET @Offset", d.Pagination())
	assert.Equal(t, "INTEGER PRIMARY KEY AUTOINCREMENT", d.AutoIncrement())
	assert.Equal(t, "CURRENT_TIMESTAMP", d.Now())
	assert.Equal(t, "TIMESTAMP", d.TimestampType())
	assert.Equal(t, "TEXT", d.StringType(255))
	assert.Equal(t, "TEXT", d.VarcharType(255))
	assert.Empty(t, d.LastInsertIDQuery())
	assert.True(t, d.SupportsLastInsertID())

	assert.Equal(t, "DROP TABLE IF EXISTS Users", d.DropTableIfExists("Users"))
	assert.Equal(t, "CREATE INDEX IF NOT EXISTS idx_users_mail ON Users(Mail)", d.CreateIndexIfNotExists("idx_users_mail", "Users", "Mail"))
}
