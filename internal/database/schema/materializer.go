package schema

import (
	"context"
	"fmt"
	"strings"

	"catgoose/dothog/internal/logger"
	dialect "github.com/catgoose/chuck"

	"github.com/jmoiron/sqlx"
)

// Materializer owns the registered table set and projects it onto the bound
// *sqlx.DB and dialect through Init/Ensure/Seed/Validate operations. It is
// intentionally narrow: it does not expose the pool, and repositories take
// *sqlx.DB directly rather than depending on a "manager" object.
type Materializer struct {
	db      *sqlx.DB
	dialect dialect.Dialect
	tables  []*TableDef
}

// NewMaterializer binds the DB pool, dialect, and the table set used by
// Init/Ensure/Seed/Validate.
func NewMaterializer(db *sqlx.DB, d dialect.Dialect, tables ...*TableDef) *Materializer {
	return &Materializer{
		db:      db,
		dialect: d,
		tables:  tables,
	}
}

// InitSchema drops and recreates all registered tables. Destructive: wipes existing data.
func (m *Materializer) InitSchema(ctx context.Context) error {
	log := logger.WithContext(ctx)
	log.Info("Initializing database schema (destructive)")

	for i := len(m.tables) - 1; i >= 0; i-- {
		stmt := m.tables[i].DropSQL(m.dialect)
		if _, err := m.db.ExecContext(ctx, stmt); err != nil {
			log.Warn("Failed to drop table (may not exist)", "table", m.tables[i].Name, "error", err)
		}
	}

	for _, td := range m.tables {
		if err := m.createTable(ctx, td); err != nil {
			return fmt.Errorf("failed to create %s table: %w", td.Name, err)
		}
	}

	log.Info("Database schema initialized successfully")
	return nil
}

// EnsureSchema is the non-destructive counterpart to InitSchema; existing tables are left intact.
func (m *Materializer) EnsureSchema(ctx context.Context) error {
	log := logger.WithContext(ctx)
	log.Info("Ensuring database schema")

	for _, td := range m.tables {
		for _, stmt := range td.CreateIfNotExistsSQL(m.dialect) {
			if _, err := m.db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("failed to ensure %s: %w", td.Name, err)
			}
		}
	}

	log.Info("Database schema ensured successfully")
	return nil
}

// SeedSchema applies WithSeedRows data via INSERT OR IGNORE; idempotent and safe on every startup.
func (m *Materializer) SeedSchema(ctx context.Context) error {
	log := logger.WithContext(ctx)

	seeded := 0
	for _, td := range m.tables {
		for _, stmt := range td.SeedSQL(m.dialect) {
			if _, err := m.db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("failed to seed %s: %w", td.Name, err)
			}
			seeded++
		}
	}

	if seeded > 0 {
		log.Info("Seed data applied", "rows", seeded)
	}
	return nil
}

// Mismatch describes a single schema mismatch; one or more are wrapped by ValidationError.
type Mismatch struct {
	Table   string
	Column  string
	Message string
}

func (e Mismatch) Error() string {
	if e.Column != "" {
		return fmt.Sprintf("%s.%s: %s", e.Table, e.Column, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Table, e.Message)
}

// ValidationError wraps one or more Mismatchs returned by ValidateSchema.
type ValidationError struct {
	Errors []Mismatch
}

func (e *ValidationError) Error() string {
	msgs := make([]string, len(e.Errors))
	for i, se := range e.Errors {
		msgs[i] = se.Error()
	}
	return fmt.Sprintf("schema validation failed (%d errors): %s", len(e.Errors), strings.Join(msgs, "; "))
}

// ValidateSchema confirms every registered table exists with all expected columns;
// returns *ValidationError on mismatch, nil when valid.
func (m *Materializer) ValidateSchema(ctx context.Context) error {
	log := logger.WithContext(ctx)
	log.Info("Validating database schema")

	var errs []Mismatch

	for _, td := range m.tables {
		var name string
		err := m.db.GetContext(ctx, &name, m.dialect.TableExistsQuery(), td.Name)
		if err != nil {
			errs = append(errs, Mismatch{Table: td.Name, Message: "table does not exist"})
			continue
		}

		var dbCols []string
		err = m.db.SelectContext(ctx, &dbCols, m.dialect.TableColumnsQuery(), td.Name)
		if err != nil {
			errs = append(errs, Mismatch{Table: td.Name, Message: fmt.Sprintf("failed to query columns: %v", err)})
			continue
		}

		dbColSet := make(map[string]bool, len(dbCols))
		for _, c := range dbCols {
			dbColSet[c] = true
		}

		for _, col := range td.SelectColumns() {
			if !dbColSet[col] {
				errs = append(errs, Mismatch{Table: td.Name, Column: col, Message: "column missing"})
			}
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}

	log.Info("Database schema validation passed")
	return nil
}

func (m *Materializer) createTable(ctx context.Context, td *TableDef) error {
	log := logger.WithContext(ctx)
	log.Info("Creating table", "table", td.Name)

	for _, stmt := range td.CreateSQL(m.dialect) {
		if _, err := m.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("create table %s: %w", td.Name, err)
		}
	}
	return nil
}
