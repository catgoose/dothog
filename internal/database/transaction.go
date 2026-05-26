package database

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// WithTransaction runs fn inside a transaction on db using the caller's
// context: commits on fn's nil return, rolls back otherwise. The deferred
// Rollback is the safety net — on any error path (Begin/fn/Commit failure
// or panic) it runs; after a successful Commit it is a no-op (Rollback on a
// committed tx returns sql.ErrTxDone which is discarded). Timeout policy is
// the caller's job — pass a derived ctx if one is needed.
func WithTransaction(ctx context.Context, db *sqlx.DB, fn func(ctx context.Context, tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := fn(ctx, tx); err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
