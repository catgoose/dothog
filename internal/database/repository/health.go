// setup:feature:database

package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// HealthCheck pings the database to check connectivity
func HealthCheck(ctx context.Context, db *sqlx.DB) error {
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	return nil
}

// CheckConnection checks if the database connection is alive
// Returns true if connection is healthy, false otherwise
func CheckConnection(ctx context.Context, db *sqlx.DB) (bool, error) {
	if err := db.PingContext(ctx); err != nil {
		return false, fmt.Errorf("database connection check failed: %w", err)
	}
	return true, nil
}
