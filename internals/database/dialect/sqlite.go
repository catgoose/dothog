// setup:feature:database

package dialect

import "fmt"

// SQLiteDialect implements Dialect for SQLite.
type SQLiteDialect struct{}

func (SQLiteDialect) Engine() Engine { return SQLite }

func (SQLiteDialect) Pagination() string {
	return "LIMIT @Limit OFFSET @Offset"
}

func (SQLiteDialect) AutoIncrement() string {
	return "INTEGER PRIMARY KEY AUTOINCREMENT"
}

func (SQLiteDialect) Now() string { return "CURRENT_TIMESTAMP" }

func (SQLiteDialect) TimestampType() string { return "TIMESTAMP" }

func (SQLiteDialect) StringType(_ int) string { return "TEXT" }

func (SQLiteDialect) VarcharType(_ int) string { return "TEXT" }

func (SQLiteDialect) DropTableIfExists(table string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", table)
}

func (SQLiteDialect) CreateIndexIfNotExists(indexName, table, columns string) string {
	return fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)", indexName, table, columns)
}

func (SQLiteDialect) LastInsertIDQuery() string { return "" }

func (SQLiteDialect) SupportsLastInsertID() bool { return true }
