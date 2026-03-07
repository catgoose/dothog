// setup:feature:database

// Package repository provides data access layer functionality.
// It includes database operations with transaction support and error handling.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"catgoose/go-htmx-demo/internals/database/dialect"
	"catgoose/go-htmx-demo/internals/database/schema"
	"catgoose/go-htmx-demo/internals/logger"

	"github.com/jmoiron/sqlx"
)

// RepoManager manages all repository access to the database.
type RepoManager struct {
	db      *sqlx.DB
	dialect dialect.Dialect
}

// NewManager creates a new RepoManager instance.
func NewManager(db *sqlx.DB, d dialect.Dialect) *RepoManager {
	return &RepoManager{
		db:      db,
		dialect: d,
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
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
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

// InitSchema initializes all database tables. Destructive: drops existing tables and recreates them, wiping data.
func (r *RepoManager) InitSchema(ctx context.Context) error {
	log := logger.WithContext(ctx)
	log.Info("Initializing database schema")

	if err := r.dropAllTables(ctx); err != nil {
		log.Info("Failed to drop existing tables (tables may not exist)", "error", err)
	}

	if err := r.createUsersTable(ctx); err != nil {
		return fmt.Errorf("failed to create Users table: %w", err)
	}

	log.Info("Database schema initialized successfully")
	return nil
}

func (r *RepoManager) dropAllTables(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, schema.UsersTable.DropSQL(r.dialect))
	return err
}

func (r *RepoManager) createUsersTable(ctx context.Context) error {
	log := logger.WithContext(ctx)
	log.Info("Creating Users table")

	for _, stmt := range schema.UsersTable.CreateSQL(r.dialect) {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	log.Info("Users table created successfully")
	return nil
}
