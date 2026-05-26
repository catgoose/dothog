package schema

import (
	"context"
	"errors"
	"testing"

	dialect "github.com/catgoose/chuck"

	_ "github.com/catgoose/chuck/driver/sqlite"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openSQLiteInMemory(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestValidateSchema_Valid(t *testing.T) {
	db := openSQLiteInMemory(t)
	d := dialect.SQLiteDialect{}

	table := NewTable("Items").
		Columns(
			AutoIncrCol("ID"),
			Col("Name", TypeString(255)).NotNull(),
		).
		WithTimestamps()

	mgr := NewMaterializer(db, d, table)
	ctx := context.Background()

	require.NoError(t, mgr.InitSchema(ctx))
	require.NoError(t, mgr.ValidateSchema(ctx))
}

func TestValidateSchema_MissingTable(t *testing.T) {
	db := openSQLiteInMemory(t)
	d := dialect.SQLiteDialect{}

	table := NewTable("NonExistent").
		Columns(AutoIncrCol("ID"))

	mgr := NewMaterializer(db, d, table)
	ctx := context.Background()

	err := mgr.ValidateSchema(ctx)
	require.Error(t, err)

	var schemaErr *ValidationError
	require.True(t, errors.As(err, &schemaErr))
	require.Len(t, schemaErr.Errors, 1)
	assert.Equal(t, "NonExistent", schemaErr.Errors[0].Table)
	assert.Contains(t, schemaErr.Errors[0].Message, "table does not exist")
}

func TestValidateSchema_MissingColumn(t *testing.T) {
	db := openSQLiteInMemory(t)
	d := dialect.SQLiteDialect{}

	actual := NewTable("Items").
		Columns(AutoIncrCol("ID"))

	mgr := NewMaterializer(db, d, actual)
	ctx := context.Background()
	require.NoError(t, mgr.InitSchema(ctx))

	expected := NewTable("Items").
		Columns(
			AutoIncrCol("ID"),
			Col("Name", TypeString(255)).NotNull(),
		)

	mgr2 := NewMaterializer(db, d, expected)
	err := mgr2.ValidateSchema(ctx)
	require.Error(t, err)

	var schemaErr *ValidationError
	require.True(t, errors.As(err, &schemaErr))
	require.Len(t, schemaErr.Errors, 1)
	assert.Equal(t, "Items", schemaErr.Errors[0].Table)
	assert.Equal(t, "Name", schemaErr.Errors[0].Column)
	assert.Contains(t, schemaErr.Errors[0].Message, "column missing")
}

func TestValidateSchema_MultipleTables(t *testing.T) {
	db := openSQLiteInMemory(t)
	d := dialect.SQLiteDialect{}

	users := NewTable("Users").
		Columns(AutoIncrCol("ID"), Col("Email", TypeString(255)))

	orders := NewTable("Orders").
		Columns(AutoIncrCol("ID"), Col("Total", TypeInt()))

	mgr := NewMaterializer(db, d, users, orders)
	ctx := context.Background()

	require.NoError(t, mgr.InitSchema(ctx))
	require.NoError(t, mgr.ValidateSchema(ctx))
}

func TestValidateSchema_ErrorString(t *testing.T) {
	err := &ValidationError{
		Errors: []Mismatch{
			{Table: "Users", Message: "table does not exist"},
			{Table: "Orders", Column: "Total", Message: "column missing"},
		},
	}
	s := err.Error()
	assert.Contains(t, s, "2 errors")
	assert.Contains(t, s, "Users: table does not exist")
	assert.Contains(t, s, "Orders.Total: column missing")
}
