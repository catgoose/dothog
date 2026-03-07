package schema

import (
	"fmt"
	"strings"

	"catgoose/go-htmx-demo/internals/database/dialect"
)

// TableDef defines a table schema.
type TableDef struct {
	Name          string
	cols          []ColumnDef
	indexes       []IndexDef
	hasSoftDelete bool
}

// NewTable creates a new table definition.
func NewTable(name string) *TableDef {
	return &TableDef{Name: name}
}

// Columns appends column definitions.
func (t *TableDef) Columns(cols ...ColumnDef) *TableDef {
	t.cols = append(t.cols, cols...)
	return t
}

// WithTimestamps appends CreatedAt and UpdatedAt columns.
func (t *TableDef) WithTimestamps() *TableDef {
	t.cols = append(t.cols, TimestampColumnDefs()...)
	return t
}

// WithSoftDelete appends a DeletedAt column and marks the table for soft-delete.
func (t *TableDef) WithSoftDelete() *TableDef {
	t.hasSoftDelete = true
	t.cols = append(t.cols, SoftDeleteColumnDefs()...)
	return t
}

// WithAuditTrail appends CreatedBy, UpdatedBy, and DeletedBy columns.
func (t *TableDef) WithAuditTrail() *TableDef {
	t.cols = append(t.cols, AuditColumnDefs()...)
	return t
}

// Indexes appends index definitions.
func (t *TableDef) Indexes(indexes ...IndexDef) *TableDef {
	t.indexes = append(t.indexes, indexes...)
	return t
}

// SelectColumns returns all column names.
func (t *TableDef) SelectColumns() []string {
	names := make([]string, len(t.cols))
	for i, c := range t.cols {
		names[i] = c.name
	}
	return names
}

// InsertColumns returns column names excluding auto-increment columns.
func (t *TableDef) InsertColumns() []string {
	var names []string
	for _, c := range t.cols {
		if !c.autoIncr {
			names = append(names, c.name)
		}
	}
	return names
}

// UpdateColumns returns only mutable column names.
func (t *TableDef) UpdateColumns() []string {
	var names []string
	for _, c := range t.cols {
		if c.mutable {
			names = append(names, c.name)
		}
	}
	return names
}

// HasSoftDelete reports whether the table uses soft-delete.
func (t *TableDef) HasSoftDelete() bool {
	return t.hasSoftDelete
}

// CreateSQL returns the CREATE TABLE statement followed by CREATE INDEX statements.
func (t *TableDef) CreateSQL(d dialect.Dialect) []string {
	var colLines []string
	for _, c := range t.cols {
		colLines = append(colLines, "\t\t\t"+c.ddl(d))
	}

	create := fmt.Sprintf("\n\t\tCREATE TABLE %s (\n%s\n\t\t)",
		t.Name, strings.Join(colLines, ",\n"))

	stmts := []string{create}
	for _, idx := range t.indexes {
		stmts = append(stmts, d.CreateIndexIfNotExists(idx.name, t.Name, idx.columns))
	}
	return stmts
}

// DropSQL returns the DROP TABLE statement for the given dialect.
func (t *TableDef) DropSQL(d dialect.Dialect) string {
	return d.DropTableIfExists(t.Name)
}
