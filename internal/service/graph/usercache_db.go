// setup:feature:graph

package graph

import (
	"context"
	"fmt"
	"time"

	"catgoose/dothog/internal/database"
	"catgoose/dothog/internal/database/schema"
	"catgoose/dothog/internal/logger"

	_ "github.com/catgoose/chuck/driver/sqlite" // Register SQLite driver
	"github.com/jmoiron/sqlx"
)

// OpenUserCacheDB opens the in-memory SQLite handle that backs the Graph
// user cache: single-connection, WAL, 30s busy timeout, with UsersTable's
// schema materialized at open time so callers get a ready-to-use DB.
func OpenUserCacheDB() (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open in-memory SQLite database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := configureUserCachePerformance(db); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			logger.Get().Error("Failed to close DB after PRAGMA error", "close_error", closeErr, "config_error", err)
		}
		return nil, fmt.Errorf("failed to configure SQLite performance: %w", err)
	}

	if err := materializeUsersTable(context.Background(), db); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			logger.Get().Error("Failed to close DB after schema error", "close_error", closeErr, "schema_error", err)
		}
		return nil, fmt.Errorf("failed to materialize user cache schema: %w", err)
	}

	return db, nil
}

func configureUserCachePerformance(db *sqlx.DB) error {
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=30000"); err != nil {
		return fmt.Errorf("failed to set busy timeout: %w", err)
	}
	return nil
}

// materializeUsersTable runs schema.Materializer.EnsureSchema with the
// graph-owned UsersTable. Both OpenUserCacheDB and UserCache.EnsureSchema
// route through this so the cache schema has exactly one source of truth.
func materializeUsersTable(ctx context.Context, db *sqlx.DB) error {
	d, err := database.SQLiteDialect()
	if err != nil {
		return fmt.Errorf("failed to create SQLite dialect: %w", err)
	}
	mat := schema.NewMaterializer(db, d, UsersTable)
	return mat.EnsureSchema(ctx)
}
