// Package database provides framework-internal database helpers.
// App databases should use chuck.OpenURL() directly.
// OpenSQLite is for framework-internal stores (error traces, session settings)
// that are always SQLite regardless of the app's primary database; this file
// is intentionally not gated by setup:feature:database so always-on stores
// like the error-trace SQLite remain wired in derived apps.
package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	dialect "github.com/catgoose/chuck"
	"github.com/jmoiron/sqlx"
)

// SQLiteDialect constructs a chuck SQLite dialect for the always-on framework
// SQLite stores (session settings, error traces, graph user cache). Callers
// reach for this instead of importing chuck directly so the chuck dialect
// import stays scoped to the app-data feature in main.go.
func SQLiteDialect() (dialect.Dialect, error) {
	return dialect.New(dialect.SQLite)
}

// OpenSQLite opens a SQLite database at the given path with standard settings.
// Used for framework-internal stores (error traces, session settings) that are
// always SQLite regardless of the app's primary database.
func OpenSQLite(ctx context.Context, dbPath string) (*sqlx.DB, error) {
	if dbPath != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	db, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=30000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	return db, nil
}
