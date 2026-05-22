// setup:feature:database

package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// HealthCheck pings the database and wraps any failure with a healthcheck context.
func HealthCheck(ctx context.Context, db *sqlx.DB) error {
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	return nil
}

// CheckConnection is the (bool, error) variant of HealthCheck for callers
// (eg. /health) that want to render a status without unwrapping the error.
func CheckConnection(ctx context.Context, db *sqlx.DB) (bool, error) {
	if err := db.PingContext(ctx); err != nil {
		return false, fmt.Errorf("database connection check failed: %w", err)
	}
	return true, nil
}
