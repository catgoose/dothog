// setup:feature:mssql

package dialect

import "fmt"

// MSSQLDialect implements Dialect for Microsoft SQL Server.
type MSSQLDialect struct{}

func (MSSQLDialect) Engine() Engine { return MSSQL }

func (MSSQLDialect) Pagination() string {
	return "OFFSET @Offset ROWS FETCH NEXT @Limit ROWS ONLY"
}

func (MSSQLDialect) AutoIncrement() string {
	return "INT PRIMARY KEY IDENTITY(1,1)"
}

func (MSSQLDialect) Now() string { return "GETDATE()" }

func (MSSQLDialect) TimestampType() string { return "DATETIME" }

func (MSSQLDialect) StringType(maxLen int) string {
	return fmt.Sprintf("NVARCHAR(%d)", maxLen)
}

func (MSSQLDialect) VarcharType(maxLen int) string {
	return fmt.Sprintf("VARCHAR(%d)", maxLen)
}

func (MSSQLDialect) DropTableIfExists(table string) string {
	return fmt.Sprintf(
		"IF EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[%s]') AND type in (N'U')) BEGIN DROP TABLE [dbo].[%s]; END",
		table, table,
	)
}

func (MSSQLDialect) CreateIndexIfNotExists(indexName, table, columns string) string {
	return fmt.Sprintf(
		"IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = '%s' AND object_id = OBJECT_ID('%s')) CREATE INDEX %s ON %s(%s)",
		indexName, table, indexName, table, columns,
	)
}

func (MSSQLDialect) LastInsertIDQuery() string { return "SELECT SCOPE_IDENTITY() AS ID" }

func (MSSQLDialect) SupportsLastInsertID() bool { return false }
