// setup:feature:database
package repository

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColumns(t *testing.T) {
	assert.Equal(t, "ID, Name, Email", Columns("ID", "Name", "Email"))
	assert.Equal(t, "ID", Columns("ID"))
	assert.Empty(t, Columns())
}

func TestPlaceholders(t *testing.T) {
	assert.Equal(t, "@ID, @Name, @Email", Placeholders("ID", "Name", "Email"))
	assert.Equal(t, "@ID", Placeholders("ID"))
	assert.Empty(t, Placeholders())
}

func TestSetClause(t *testing.T) {
	assert.Equal(t, "Name = @Name, Email = @Email", SetClause("Name", "Email"))
	assert.Equal(t, "Name = @Name", SetClause("Name"))
	assert.Empty(t, SetClause())
}

func TestInsertInto(t *testing.T) {
	stmt := InsertInto("Users", "Name", "Email")
	assert.Equal(t, "INSERT INTO Users (Name, Email) VALUES (@Name, @Email)", stmt)
}

func TestInsertInto_SingleColumn(t *testing.T) {
	stmt := InsertInto("Logs", "Message")
	assert.Equal(t, "INSERT INTO Logs (Message) VALUES (@Message)", stmt)
}

func TestNamedArgs(t *testing.T) {
	args := NamedArgs(map[string]any{
		"Name":  "Alice",
		"Email": "alice@example.com",
	})
	assert.Len(t, args, 2)
	for _, a := range args {
		na, ok := a.(sql.NamedArg)
		assert.True(t, ok)
		assert.NotEmpty(t, na.Name)
	}
}

func TestNamedArgs_Empty(t *testing.T) {
	args := NamedArgs(map[string]any{})
	assert.Empty(t, args)
}
