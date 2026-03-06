// setup:feature:database

// Package database provides database connection management and data access utilities.
package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"catgoose/go-htmx-demo/internals/database/dialect"

	"github.com/catgoose/dio"
	"github.com/jmoiron/sqlx"
)

// Open establishes a database connection based on the engine type.
// For SQLite, it reads DB_PATH from the environment (defaults to "db/app.db").
// setup:feature:mssql:start
// For MSSQL, it reads DB_HOST, DB_DATABASE, DB_USER, DB_PASSWORD from the environment.
// setup:feature:mssql:end
func Open(ctx context.Context, engine dialect.Engine) (*sqlx.DB, error) {
	switch engine {
	case dialect.SQLite:
		return openSQLiteDB(ctx)
	// setup:feature:mssql:start
	case dialect.MSSQL:
		return openMSSQLDB(ctx)
	// setup:feature:mssql:end
	default:
		return nil, fmt.Errorf("unsupported database engine: %q", engine)
	}
}

func openSQLiteDB(ctx context.Context) (*sqlx.DB, error) {
	dbPath := "db/app.db"
	if path, err := dio.Env("DB_PATH"); err == nil && path != "" {
		dbPath = path
	}

	if dbPath != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	db, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// SQLite has limited concurrency
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=30000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	return db, nil
}

// Ping pings the database to check connectivity
func Ping(ctx context.Context, db *sqlx.DB) error {
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	return nil
}
