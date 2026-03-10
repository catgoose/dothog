package schema

import (
	"fmt"
	"strings"

	"catgoose/go-htmx-demo/internals/database/dialect"
)

// UniqueConstraint defines a composite UNIQUE constraint across multiple columns.
type UniqueConstraint struct {
	columns []string
}

// ddl renders the constraint as a DDL fragment.
func (uc UniqueConstraint) ddl() string {
	return fmt.Sprintf("UNIQUE (%s)", strings.Join(uc.columns, ", "))
}

// SeedRow represents a row of seed data as column name → value pairs.
type SeedRow map[string]string

// TableDef defines a table schema.
type TableDef struct {
	Name              string
	cols              []ColumnDef
	indexes           []IndexDef
	uniqueConstraints []UniqueConstraint
	seedRows          []SeedRow
	hasSoftDelete     bool
	hasVersion        bool
	hasExpiry         bool
	hasArchive        bool
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

// WithVersion appends a Version column for optimistic concurrency control.
func (t *TableDef) WithVersion() *TableDef {
	t.hasVersion = true
	t.cols = append(t.cols, VersionColumnDefs()...)
	return t
}

// WithSortOrder appends a SortOrder column for manual ordering.
func (t *TableDef) WithSortOrder() *TableDef {
	t.cols = append(t.cols, SortOrderColumnDefs()...)
	return t
}

// WithStatus appends a Status column with the given default value.
func (t *TableDef) WithStatus(defaultStatus string) *TableDef {
	t.cols = append(t.cols, StatusColumnDefs(defaultStatus)...)
	return t
}

// WithNotes appends a nullable Notes text column.
func (t *TableDef) WithNotes() *TableDef {
	t.cols = append(t.cols, NotesColumnDefs()...)
	return t
}

// WithUUID appends a UUID column (NOT NULL, UNIQUE, immutable).
func (t *TableDef) WithUUID() *TableDef {
	t.cols = append(t.cols, UUIDColumnDefs()...)
	return t
}

// WithParent appends a nullable ParentID column for tree/hierarchy structures.
func (t *TableDef) WithParent() *TableDef {
	t.cols = append(t.cols, ParentColumnDefs()...)
	return t
}

// WithReplacement appends a nullable ReplacedByID column for entity lineage tracking.
func (t *TableDef) WithReplacement() *TableDef {
	t.cols = append(t.cols, ReplacementColumnDefs()...)
	return t
}

// WithArchive appends a nullable ArchivedAt timestamp column.
func (t *TableDef) WithArchive() *TableDef {
	t.hasArchive = true
	t.cols = append(t.cols, ArchiveColumnDefs()...)
	return t
}

// WithExpiry appends a nullable ExpiresAt timestamp column.
func (t *TableDef) WithExpiry() *TableDef {
	t.hasExpiry = true
	t.cols = append(t.cols, ExpiryColumnDefs()...)
	return t
}

// UniqueColumns adds a composite UNIQUE constraint across the given columns.
func (t *TableDef) UniqueColumns(columns ...string) *TableDef {
	t.uniqueConstraints = append(t.uniqueConstraints, UniqueConstraint{columns: columns})
	return t
}

// Indexes appends index definitions.
func (t *TableDef) Indexes(indexes ...IndexDef) *TableDef {
	t.indexes = append(t.indexes, indexes...)
	return t
}

// WithSeedRows declares initial seed data for the table.
// Each SeedRow maps column names to literal SQL values (strings should be single-quoted).
func (t *TableDef) WithSeedRows(rows ...SeedRow) *TableDef {
	t.seedRows = append(t.seedRows, rows...)
	return t
}

// SeedRows returns the declared seed data.
func (t *TableDef) SeedRows() []SeedRow {
	return t.seedRows
}

// HasSeedData reports whether any seed rows have been declared.
func (t *TableDef) HasSeedData() bool {
	return len(t.seedRows) > 0
}

// SeedSQL returns INSERT statements for all seed rows using the table's insert columns.
// Values are inserted using INSERT OR IGNORE (SQLite) / IF NOT EXISTS style to be idempotent.
func (t *TableDef) SeedSQL() []string {
	if len(t.seedRows) == 0 {
		return nil
	}

	insertCols := t.InsertColumns()
	var stmts []string
	for _, row := range t.seedRows {
		var vals []string
		for _, col := range insertCols {
			if v, ok := row[col]; ok {
				vals = append(vals, v)
			} else {
				vals = append(vals, "NULL")
			}
		}
		stmt := fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES (%s)",
			t.Name,
			strings.Join(insertCols, ", "),
			strings.Join(vals, ", "),
		)
		stmts = append(stmts, stmt)
	}
	return stmts
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

// HasVersion reports whether the table uses optimistic concurrency control.
func (t *TableDef) HasVersion() bool {
	return t.hasVersion
}

// HasExpiry reports whether the table uses expiry.
func (t *TableDef) HasExpiry() bool {
	return t.hasExpiry
}

// HasArchive reports whether the table uses archive.
func (t *TableDef) HasArchive() bool {
	return t.hasArchive
}

// columnBody returns the formatted column definitions for use in CREATE TABLE statements.
func (t *TableDef) columnBody(d dialect.Dialect) string {
	var colLines []string
	for _, c := range t.cols {
		colLines = append(colLines, "\t\t\t"+c.ddl(d))
	}
	if len(t.uniqueConstraints) > 0 {
		for _, uc := range t.uniqueConstraints {
			colLines = append(colLines, "\t\t\t"+uc.ddl())
		}
	}
	return strings.Join(colLines, ",\n")
}

// CreateSQL returns the CREATE TABLE statement followed by CREATE INDEX statements.
func (t *TableDef) CreateSQL(d dialect.Dialect) []string {
	create := fmt.Sprintf("\n\t\tCREATE TABLE %s (\n%s\n\t\t)",
		t.Name, t.columnBody(d))

	stmts := []string{create}
	for _, idx := range t.indexes {
		stmts = append(stmts, d.CreateIndexIfNotExists(idx.name, t.Name, idx.columns))
	}
	return stmts
}

// CreateIfNotExistsSQL returns CREATE TABLE IF NOT EXISTS followed by CREATE INDEX IF NOT EXISTS statements.
func (t *TableDef) CreateIfNotExistsSQL(d dialect.Dialect) []string {
	create := d.CreateTableIfNotExists(t.Name, t.columnBody(d))

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
