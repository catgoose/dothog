// setup:feature:database

// Package repository provides data access layer functionality.
// It includes database operations with transaction support and error handling.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"catgoose/go-htmx-demo/internals/logger"

	"github.com/jmoiron/sqlx"
)

// RepoManager manages all repository access to the database.
type RepoManager struct {
	db *sqlx.DB
}

// NewManager creates a new RepoManager instance.
func NewManager(db *sqlx.DB) *RepoManager {
	return &RepoManager{
		db: db,
	}
}

// GetDB returns the database connection
func (r *RepoManager) GetDB() *sqlx.DB {
	return r.db
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
	dropSQL := `
		IF EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[Users]') AND type in (N'U'))
		BEGIN
			DROP TABLE [dbo].[Users];
		END
	`
	_, err := r.db.ExecContext(ctx, dropSQL)
	return err
}

func (r *RepoManager) createUsersTable(ctx context.Context) error {
	log := logger.WithContext(ctx)
	log.Info("Creating Users table")

	createSQL := `
		CREATE TABLE Users (
			ID INT PRIMARY KEY IDENTITY(1,1),
			AzureId VARCHAR(255) NOT NULL UNIQUE,
			GivenName NVARCHAR(255),
			Surname NVARCHAR(255),
			DisplayName NVARCHAR(255),
			UserPrincipalName NVARCHAR(255) NOT NULL,
			Mail NVARCHAR(255),
			JobTitle NVARCHAR(255),
			OfficeLocation NVARCHAR(255),
			Department NVARCHAR(255),
			CompanyName NVARCHAR(255),
			AccountName NVARCHAR(255),
			LastLoginAt DATETIME,
			CreatedAt DATETIME NOT NULL DEFAULT GETDATE(),
			UpdatedAt DATETIME NOT NULL DEFAULT GETDATE()
		);
		CREATE INDEX idx_users_azureid ON Users(AzureId);
		CREATE INDEX idx_users_userprincipalname ON Users(UserPrincipalName);
		CREATE INDEX idx_users_displayname ON Users(DisplayName);
		CREATE INDEX idx_users_mail ON Users(Mail);
		CREATE INDEX idx_users_lastloginat ON Users(LastLoginAt);
	`
	_, err := r.db.ExecContext(ctx, createSQL)
	if err != nil {
		return err
	}
	log.Info("Users table created successfully")
	return nil
}
