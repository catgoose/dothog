// setup:feature:database

// Package dialect provides database engine abstractions for composable SQL fragments.
// It allows switching between database engines (e.g., MSSQL for production, SQLite for development)
// while keeping SQL visible and explicit.
package dialect

import "fmt"

// Engine identifies a database engine.
type Engine string

const (
	MSSQL  Engine = "sqlserver"
	SQLite Engine = "sqlite3"
)

// ParseEngine converts a string to an Engine, returning an error for unknown values.
func ParseEngine(s string) (Engine, error) {
	switch s {
	case "sqlserver", "mssql":
		return MSSQL, nil
	case "sqlite3", "sqlite":
		return SQLite, nil
	default:
		return "", fmt.Errorf("unknown database engine: %q (expected sqlserver, mssql, sqlite3, or sqlite)", s)
	}
}

// Dialect provides engine-specific SQL fragments.
// Implementations return raw SQL strings that callers compose into full queries.
type Dialect interface {
	// Engine returns the engine identifier (used as the driver name for sqlx.Open).
	Engine() Engine

	// Pagination returns the pagination clause for the engine.
	//   MSSQL:  "OFFSET @Offset ROWS FETCH NEXT @Limit ROWS ONLY"
	//   SQLite: "LIMIT @Limit OFFSET @Offset"
	Pagination() string

	// AutoIncrement returns the column definition fragment for an auto-incrementing primary key.
	//   MSSQL:  "INT PRIMARY KEY IDENTITY(1,1)"
	//   SQLite: "INTEGER PRIMARY KEY AUTOINCREMENT"
	AutoIncrement() string

	// Now returns the SQL expression for the current timestamp.
	//   MSSQL:  "GETDATE()"
	//   SQLite: "CURRENT_TIMESTAMP"
	Now() string

	// TimestampType returns the column type for timestamps.
	//   MSSQL:  "DATETIME"
	//   SQLite: "TIMESTAMP"
	TimestampType() string

	// StringType returns the column type for a string with the given max length.
	//   MSSQL:  "NVARCHAR(255)"
	//   SQLite: "TEXT"
	StringType(maxLen int) string

	// VarcharType returns the column type for a varchar with the given max length.
	//   MSSQL:  "VARCHAR(255)"
	//   SQLite: "TEXT"
	VarcharType(maxLen int) string

	// DropTableIfExists returns the statement to drop a table if it exists.
	//   MSSQL:  "IF EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[Users]') ...) BEGIN DROP TABLE [dbo].[Users]; END"
	//   SQLite: "DROP TABLE IF EXISTS Users"
	DropTableIfExists(table string) string

	// CreateIndexIfNotExists returns the statement to create an index if it doesn't exist.
	//   MSSQL:  "IF NOT EXISTS (SELECT * FROM sys.indexes ...) CREATE INDEX ..."
	//   SQLite: "CREATE INDEX IF NOT EXISTS ..."
	CreateIndexIfNotExists(indexName, table, columns string) string

	// LastInsertIDQuery returns SQL to retrieve the last inserted ID, or empty string
	// if the driver supports Result.LastInsertId() natively.
	//   MSSQL:  "SELECT SCOPE_IDENTITY() AS ID"
	//   SQLite: "" (use result.LastInsertId())
	LastInsertIDQuery() string

	// SupportsLastInsertID reports whether the driver supports Result.LastInsertId().
	//   MSSQL:  false (use SCOPE_IDENTITY() query)
	//   SQLite: true
	SupportsLastInsertID() bool
}

// New returns a Dialect for the given engine.
func New(engine Engine) (Dialect, error) {
	switch engine {
	// setup:feature:mssql:start
	case MSSQL:
		return MSSQLDialect{}, nil
	// setup:feature:mssql:end
	case SQLite:
		return SQLiteDialect{}, nil
	default:
		return nil, fmt.Errorf("unsupported database engine: %q", engine)
	}
}
