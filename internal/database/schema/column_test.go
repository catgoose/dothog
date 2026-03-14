package schema

import (
	"testing"

	"catgoose/dothog/internal/database/dialect"

	"github.com/stretchr/testify/assert"
)

func TestCol_DDL_SQLite(t *testing.T) {
	d := dialect.SQLiteDialect{}

	tests := []struct {
		name string
		col  ColumnDef
		want string
	}{
		{
			name: "simple nullable column",
			col:  Col("Name", TypeString(255)),
			want: "Name TEXT",
		},
		{
			name: "not null column",
			col:  Col("Email", TypeString(255)).NotNull(),
			want: "Email TEXT NOT NULL",
		},
		{
			name: "unique column",
			col:  Col("AzureId", TypeVarchar(255)).NotNull().Unique(),
			want: "AzureId TEXT NOT NULL UNIQUE",
		},
		{
			name: "timestamp with default",
			col:  Col("CreatedAt", TypeTimestamp()).NotNull().DefaultFn(func(d dialect.Dialect) string { return d.Now() }),
			want: "CreatedAt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
		},
		{
			name: "literal default",
			col:  Col("Status", TypeString(50)).NotNull().Default("'active'"),
			want: "Status TEXT NOT NULL DEFAULT 'active'",
		},
		{
			name: "auto increment column",
			col:  AutoIncrCol("ID"),
			want: "ID INTEGER PRIMARY KEY AUTOINCREMENT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.col.ddl(d))
		})
	}
}

func TestCol_DDL_MSSQL(t *testing.T) {
	d := dialect.MSSQLDialect{}

	tests := []struct {
		name string
		col  ColumnDef
		want string
	}{
		{
			name: "string column",
			col:  Col("Name", TypeString(255)),
			want: "Name NVARCHAR(255)",
		},
		{
			name: "varchar column",
			col:  Col("AzureId", TypeVarchar(255)).NotNull().Unique(),
			want: "AzureId VARCHAR(255) NOT NULL UNIQUE",
		},
		{
			name: "timestamp with default",
			col:  Col("CreatedAt", TypeTimestamp()).NotNull().DefaultFn(func(d dialect.Dialect) string { return d.Now() }),
			want: "CreatedAt DATETIME NOT NULL DEFAULT GETDATE()",
		},
		{
			name: "auto increment column",
			col:  AutoIncrCol("ID"),
			want: "ID INT PRIMARY KEY IDENTITY(1,1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.col.ddl(d))
		})
	}
}

func TestCol_Name(t *testing.T) {
	c := Col("Email", TypeString(255))
	assert.Equal(t, "Email", c.Name())
}

func TestCol_Immutable_Mutable(t *testing.T) {
	c := Col("Name", TypeString(255))
	assert.True(t, c.mutable)

	c = c.Immutable()
	assert.False(t, c.mutable)

	c = c.Mutable()
	assert.True(t, c.mutable)
}

func TestAutoIncrCol_IsImmutable(t *testing.T) {
	c := AutoIncrCol("ID")
	assert.False(t, c.mutable)
	assert.True(t, c.pk)
	assert.True(t, c.autoIncr)
}

func TestTypeLiteral(t *testing.T) {
	fn := TypeLiteral("BLOB")
	assert.Equal(t, "BLOB", fn(dialect.SQLiteDialect{}))
	assert.Equal(t, "BLOB", fn(dialect.MSSQLDialect{}))
}

func TestTypeInt(t *testing.T) {
	fn := TypeInt()
	assert.Equal(t, "INTEGER", fn(dialect.SQLiteDialect{}))
	assert.Equal(t, "INT", fn(dialect.MSSQLDialect{}))
}

func TestTypeText(t *testing.T) {
	fn := TypeText()
	assert.Equal(t, "TEXT", fn(dialect.SQLiteDialect{}))
	assert.Equal(t, "NVARCHAR(MAX)", fn(dialect.MSSQLDialect{}))
}
