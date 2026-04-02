// setup:feature:database
package repository

import (
	"context"
	"errors"
	"testing"

	dialect "github.com/catgoose/chuck"
	"catgoose/dothog/internal/database/schema"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSchema_Valid(t *testing.T) {
	db := openSQLiteInMemory(t)
	d := dialect.SQLiteDialect{}

	table := schema.NewTable("Items").
		Columns(
			schema.AutoIncrCol("ID"),
			schema.Col("Name", schema.TypeString(255)).NotNull(),
		).
		WithTimestamps()

	mgr := NewManager(db, d, table)
	ctx := context.Background()

	require.NoError(t, mgr.InitSchema(ctx))
	require.NoError(t, mgr.ValidateSchema(ctx))
}

func TestValidateSchema_MissingTable(t *testing.T) {
	db := openSQLiteInMemory(t)
	d := dialect.SQLiteDialect{}

	table := schema.NewTable("NonExistent").
		Columns(schema.AutoIncrCol("ID"))

	mgr := NewManager(db, d, table)
	ctx := context.Background()

	err := mgr.ValidateSchema(ctx)
	require.Error(t, err)

	var schemaErr *SchemaValidationError
	require.True(t, errors.As(err, &schemaErr))
	require.Len(t, schemaErr.Errors, 1)
	assert.Equal(t, "NonExistent", schemaErr.Errors[0].Table)
	assert.Contains(t, schemaErr.Errors[0].Message, "table does not exist")
}

func TestValidateSchema_MissingColumn(t *testing.T) {
	db := openSQLiteInMemory(t)
	d := dialect.SQLiteDialect{}

	// Create the table with only ID.
	actual := schema.NewTable("Items").
		Columns(schema.AutoIncrCol("ID"))

	mgr := NewManager(db, d, actual)
	ctx := context.Background()
	require.NoError(t, mgr.InitSchema(ctx))

	// Now validate against a definition that expects an extra column.
	expected := schema.NewTable("Items").
		Columns(
			schema.AutoIncrCol("ID"),
			schema.Col("Name", schema.TypeString(255)).NotNull(),
		)

	mgr2 := NewManager(db, d, expected)
	err := mgr2.ValidateSchema(ctx)
	require.Error(t, err)

	var schemaErr *SchemaValidationError
	require.True(t, errors.As(err, &schemaErr))
	require.Len(t, schemaErr.Errors, 1)
	assert.Equal(t, "Items", schemaErr.Errors[0].Table)
	assert.Equal(t, "Name", schemaErr.Errors[0].Column)
	assert.Contains(t, schemaErr.Errors[0].Message, "column missing")
}

func TestValidateSchema_MultipleTables(t *testing.T) {
	db := openSQLiteInMemory(t)
	d := dialect.SQLiteDialect{}

	users := schema.NewTable("Users").
		Columns(schema.AutoIncrCol("ID"), schema.Col("Email", schema.TypeString(255)))

	orders := schema.NewTable("Orders").
		Columns(schema.AutoIncrCol("ID"), schema.Col("Total", schema.TypeInt()))

	mgr := NewManager(db, d, users, orders)
	ctx := context.Background()

	require.NoError(t, mgr.InitSchema(ctx))
	require.NoError(t, mgr.ValidateSchema(ctx))
}

func TestValidateSchema_ErrorString(t *testing.T) {
	err := &SchemaValidationError{
		Errors: []SchemaError{
			{Table: "Users", Message: "table does not exist"},
			{Table: "Orders", Column: "Total", Message: "column missing"},
		},
	}
	s := err.Error()
	assert.Contains(t, s, "2 errors")
	assert.Contains(t, s, "Users: table does not exist")
	assert.Contains(t, s, "Orders.Total: column missing")
}
