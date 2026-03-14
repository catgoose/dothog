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

func (SQLiteDialect) IntType() string { return "INTEGER" }

func (SQLiteDialect) TextType() string { return "TEXT" }

func (SQLiteDialect) CreateTableIfNotExists(table, body string) string {
	return fmt.Sprintf("\n\t\tCREATE TABLE IF NOT EXISTS %s (\n%s\n\t\t)", table, body)
}

func (SQLiteDialect) DropTableIfExists(table string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", table)
}

func (SQLiteDialect) CreateIndexIfNotExists(indexName, table, columns string) string {
	return fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)", indexName, table, columns)
}

func (SQLiteDialect) LastInsertIDQuery() string { return "" }

func (SQLiteDialect) SupportsLastInsertID() bool { return true }

func (SQLiteDialect) TableExistsQuery() string {
	return "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
}

func (SQLiteDialect) TableColumnsQuery() string {
	return "SELECT name FROM pragma_table_info(?)"
}
