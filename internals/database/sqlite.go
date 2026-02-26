// setup:feature:graph

package database

import (
	"fmt"
	"time"

	"catgoose/go-htmx-template/internals/logger"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // Register SQLite driver
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

// OpenSQLiteInMemory opens an in-memory SQLite database connection and initializes the schema
func OpenSQLiteInMemory() (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open in-memory SQLite database: %w", err)
	}

	// Configure connection pool for SQLite concurrency
	// SQLite has limited concurrency, so we use conservative settings
	db.SetMaxOpenConns(1) // Only one connection to prevent locking issues
	db.SetMaxIdleConns(1) // Keep one idle connection
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Configure SQLite for better performance and concurrency
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

// configureSQLitePerformance sets up SQLite for optimal performance and concurrency
func configureSQLitePerformance(db *sqlx.DB) error {
	// Enable WAL mode for better concurrency (multiple readers, single writer)
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set busy timeout to 30 seconds for better concurrency handling
	if _, err := db.Exec("PRAGMA busy_timeout=30000"); err != nil {
		return fmt.Errorf("failed to set busy timeout: %w", err)
	}

	return nil
}

// InitSQLiteUserCacheSchema initializes the SQLite database schema for user cache
func InitSQLiteUserCacheSchema(db *sqlx.DB) error {
	_, err := db.Exec(userCacheSchema)
	if err != nil {
		return fmt.Errorf("failed to create SQLite user cache tables: %w", err)
	}
	return nil
}
