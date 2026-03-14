// setup:feature:database

package repository

import (
	"database/sql"
	"testing"

	"catgoose/dothog/internal/database/dialect"

	"github.com/stretchr/testify/assert"
)

func TestSelectBuilder_Basic(t *testing.T) {
	query, args := NewSelect("Users", "ID", "Name", "Email").Build()
	assert.Equal(t, "SELECT ID, Name, Email FROM Users", query)
	assert.Empty(t, args)
}

func TestSelectBuilder_WithWhere(t *testing.T) {
	w := NewWhere().
		And("Status = @Status", sql.Named("Status", "active")).
		NotDeleted()

	query, args := NewSelect("Tasks", "ID", "Title").
		Where(w).
		Build()

	assert.Equal(t, "SELECT ID, Title FROM Tasks WHERE Status = @Status AND DeletedAt IS NULL", query)
	assert.Len(t, args, 1)
}

func TestSelectBuilder_WithOrderBy(t *testing.T) {
	query, _ := NewSelect("Items", "ID", "Name").
		OrderBy("Name ASC").
		Build()

	assert.Equal(t, "SELECT ID, Name FROM Items ORDER BY Name ASC", query)
}

func TestSelectBuilder_WithOrderByFromBuilder(t *testing.T) {
	colMap := map[string]string{"name": "Name", "created": "CreatedAt"}
	query, _ := NewSelect("Items", "ID", "Name").
		OrderByMap("name:desc", colMap, "CreatedAt ASC").
		Build()

	assert.Contains(t, query, "ORDER BY Name DESC")
}

func TestSelectBuilder_WithPagination(t *testing.T) {
	query, args := NewSelect("Items", "ID", "Name").
		OrderBy("ID ASC").
		Paginate(10, 20).
		Build()

	assert.Contains(t, query, "LIMIT @Limit OFFSET @Offset")
	assert.Contains(t, args, sql.Named("Limit", 10))
	assert.Contains(t, args, sql.Named("Offset", 20))
}

func TestSelectBuilder_WithDialectPagination(t *testing.T) {
	d := dialect.SQLiteDialect{}
	query, args := NewSelect("Items", "ID", "Name").
		OrderBy("ID ASC").
		Paginate(10, 20).
		WithDialect(d).
		Build()

	assert.Contains(t, query, "LIMIT @Limit OFFSET @Offset")
	assert.Contains(t, args, sql.Named("Limit", 10))
	assert.Contains(t, args, sql.Named("Offset", 20))
}

func TestSelectBuilder_CountQuery(t *testing.T) {
	w := NewWhere().NotDeleted()

	query, args := NewSelect("Tasks", "ID", "Title", "Status").
		Where(w).
		CountQuery()

	assert.Equal(t, "SELECT COUNT(*) FROM Tasks WHERE DeletedAt IS NULL", query)
	assert.Empty(t, args)
}

func TestSelectBuilder_FullComposition(t *testing.T) {
	w := NewWhere().
		NotDeleted().
		HasStatus("active")

	query, args := NewSelect("Tasks", "ID", "Title", "Status").
		Where(w).
		OrderBy("CreatedAt DESC").
		Paginate(25, 50).
		Build()

	assert.Equal(t,
		"SELECT ID, Title, Status FROM Tasks WHERE DeletedAt IS NULL AND Status = @Status ORDER BY CreatedAt DESC LIMIT @Limit OFFSET @Offset",
		query,
	)
	assert.Len(t, args, 3) // Status, Offset, Limit
}
