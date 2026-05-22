// setup:feature:graph

package database

import (
	"fmt"
	"time"

	"catgoose/dothog/internal/logger"

	_ "github.com/catgoose/chuck/driver/sqlite" // Register SQLite driver
	"github.com/jmoiron/sqlx"
)

const userCacheSchema = `
	CREATE TABLE IF NOT EXISTS Users (
		AzureId TEXT PRIMARY KEY,
		GivenName TEXT,
		Surname TEXT,
		DisplayName TEXT,
		UserPrincipalName TEXT,
		Mail TEXT,
		JobTitle TEXT,
		OfficeLocation TEXT,
		Department TEXT,
		CompanyName TEXT,
		AccountName TEXT,
		UpdatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create indexes for better performance
	CREATE INDEX IF NOT EXISTS idx_users_azureid ON Users(AzureId);
	CREATE INDEX IF NOT EXISTS idx_users_displayname ON Users(DisplayName);
	CREATE INDEX IF NOT EXISTS idx_users_mail ON Users(Mail);
	CREATE INDEX IF NOT EXISTS idx_users_updatedat ON Users(UpdatedAt);
`

// OpenSQLiteInMemory is a single-connection in-memory SQLite handle with WAL, a 30s busy timeout, and the user-cache schema applied.
func OpenSQLiteInMemory() (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open in-memory SQLite database: %w", err)
	}

	// Single connection avoids SQLite write-locking contention.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := configureSQLitePerformance(db); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			logger.Get().Error("Failed to close database connection after performance configuration error", "close_error", closeErr, "config_error", err)
		}
		return nil, fmt.Errorf("failed to configure SQLite performance: %w", err)
	}

	if err := InitSQLiteUserCacheSchema(db); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			logger.Get().Error("Failed to close database connection after schema initialization error", "close_error", closeErr, "schema_error", err)
		}
		return nil, fmt.Errorf("failed to initialize SQLite schema: %w", err)
	}

	return db, nil
}

// configureSQLitePerformance enables WAL journal mode and a 30s busy timeout.
func configureSQLitePerformance(db *sqlx.DB) error {
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA busy_timeout=30000"); err != nil {
		return fmt.Errorf("failed to set busy timeout: %w", err)
	}

	return nil
}

// InitSQLiteUserCacheSchema applies the Users-table schema (idempotent via IF NOT EXISTS).
func InitSQLiteUserCacheSchema(db *sqlx.DB) error {
	_, err := db.Exec(userCacheSchema)
	if err != nil {
		return fmt.Errorf("failed to create SQLite user cache tables: %w", err)
	}
	return nil
}
