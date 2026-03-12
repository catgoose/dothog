// setup:feature:database

// Package repository provides data access layer functionality.
// It includes database operations with transaction support and error handling.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"catgoose/dothog/internal/database/dialect"
	"catgoose/dothog/internal/database/schema"
	"catgoose/dothog/internal/logger"

	"github.com/jmoiron/sqlx"
)

// RepoManager manages all repository access to the database.
type RepoManager struct {
	db      *sqlx.DB
	dialect dialect.Dialect
	tables  []*schema.TableDef
}

// NewManager creates a new RepoManager instance.
// Tables are registered for use by InitSchema and EnsureSchema.
func NewManager(db *sqlx.DB, d dialect.Dialect, tables ...*schema.TableDef) *RepoManager {
	return &RepoManager{
		db:      db,
		dialect: d,
		tables:  tables,
	}
}

// GetDB returns the database connection
func (r *RepoManager) GetDB() *sqlx.DB {
	return r.db
}

// Dialect returns the dialect for engine-specific SQL fragments.
func (r *RepoManager) Dialect() dialect.Dialect {
	return r.dialect
}

// GetExecer is satisfied by *sqlx.DB and *sqlx.Tx for use in repo methods that accept an optional transaction.
type GetExecer interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Exec returns the transaction if non-nil, otherwise the manager's DB connection.
// This eliminates the repeated pattern of choosing between db and tx in repository methods.
func (r *RepoManager) Exec(tx *sqlx.Tx) GetExecer {
	if tx != nil {
		return tx
	}
	return r.db
}

// WithTransaction runs fn inside a transaction. On success the transaction is committed; on error it is rolled back.
func (r *RepoManager) WithTransaction(ctx context.Context, fn func(ctx context.Context, tx *sqlx.Tx) error) error {
	txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tx, err := r.db.BeginTxx(txCtx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	if err := fn(txCtx, tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Close closes the database connection
func (r *RepoManager) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// InitSchema drops and recreates all registered tables. Destructive: wipes existing data.
func (r *RepoManager) InitSchema(ctx context.Context) error {
	log := logger.WithContext(ctx)
	log.Info("Initializing database schema (destructive)")

	// Drop in reverse order to respect potential FK dependencies.
	for i := len(r.tables) - 1; i >= 0; i-- {
		stmt := r.tables[i].DropSQL(r.dialect)
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			log.Warn("Failed to drop table (may not exist)", "table", r.tables[i].Name, "error", err)
		}
	}

	for _, td := range r.tables {
		if err := r.createTable(ctx, td); err != nil {
			return fmt.Errorf("failed to create %s table: %w", td.Name, err)
		}
	}

	log.Info("Database schema initialized successfully")
	return nil
}

// EnsureSchema creates registered tables if they do not already exist. Non-destructive.
func (r *RepoManager) EnsureSchema(ctx context.Context) error {
	log := logger.WithContext(ctx)
	log.Info("Ensuring database schema")

	for _, td := range r.tables {
		for _, stmt := range td.CreateIfNotExistsSQL(r.dialect) {
			if _, err := r.db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("failed to ensure %s: %w", td.Name, err)
			}
		}
	}

	log.Info("Database schema ensured successfully")
	return nil
}

// SeedSchema inserts seed data declared on registered tables via WithSeedRows.
// Uses INSERT OR IGNORE to be idempotent — safe to run on every startup.
func (r *RepoManager) SeedSchema(ctx context.Context) error {
	log := logger.WithContext(ctx)

	seeded := 0
	for _, td := range r.tables {
		for _, stmt := range td.SeedSQL() {
			if _, err := r.db.ExecContext(ctx, stmt); err != nil {
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

// SchemaError describes a single schema validation failure.
type SchemaError struct {
	Table   string
	Column  string
	Message string
}

func (e SchemaError) Error() string {
	if e.Column != "" {
		return fmt.Sprintf("%s.%s: %s", e.Table, e.Column, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Table, e.Message)
}

// SchemaValidationError is returned by ValidateSchema when the database does not match the expected schema.
type SchemaValidationError struct {
	Errors []SchemaError
}

func (e *SchemaValidationError) Error() string {
	msgs := make([]string, len(e.Errors))
	for i, se := range e.Errors {
		msgs[i] = se.Error()
	}
	return fmt.Sprintf("schema validation failed (%d errors): %s", len(e.Errors), strings.Join(msgs, "; "))
}

// ValidateSchema checks that every registered table exists and contains all expected columns.
// Returns a *SchemaValidationError if the database schema does not match, nil if valid.
func (r *RepoManager) ValidateSchema(ctx context.Context) error {
	log := logger.WithContext(ctx)
	log.Info("Validating database schema")

	var errs []SchemaError

	for _, td := range r.tables {
		// Check table exists.
		var name string
		err := r.db.GetContext(ctx, &name, r.dialect.TableExistsQuery(), td.Name)
		if err != nil {
			errs = append(errs, SchemaError{Table: td.Name, Message: "table does not exist"})
			continue
		}

		// Get actual columns.
		var dbCols []string
		err = r.db.SelectContext(ctx, &dbCols, r.dialect.TableColumnsQuery(), td.Name)
		if err != nil {
			errs = append(errs, SchemaError{Table: td.Name, Message: fmt.Sprintf("failed to query columns: %v", err)})
			continue
		}

		dbColSet := make(map[string]bool, len(dbCols))
		for _, c := range dbCols {
			dbColSet[c] = true
		}

		// Check expected columns exist.
		for _, col := range td.SelectColumns() {
			if !dbColSet[col] {
				errs = append(errs, SchemaError{Table: td.Name, Column: col, Message: "column missing"})
			}
		}
	}

	if len(errs) > 0 {
		return &SchemaValidationError{Errors: errs}
	}

	log.Info("Database schema validation passed")
	return nil
}

func (r *RepoManager) createTable(ctx context.Context, td *schema.TableDef) error {
	log := logger.WithContext(ctx)
	log.Info("Creating table", "table", td.Name)

	for _, stmt := range td.CreateSQL(r.dialect) {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
