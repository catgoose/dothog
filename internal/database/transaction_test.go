package database

import (
	"context"
	"errors"
	"testing"

	_ "github.com/catgoose/chuck/driver/sqlite"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTxTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("CREATE TABLE Items (Id INTEGER PRIMARY KEY, Name TEXT)")
	require.NoError(t, err)
	return db
}

func itemCount(t *testing.T, db *sqlx.DB) int {
	t.Helper()
	var n int
	require.NoError(t, db.GetContext(context.Background(), &n, "SELECT COUNT(*) FROM Items"))
	return n
}

func TestWithTransaction_CommitsOnSuccess(t *testing.T) {
	db := openTxTestDB(t)

	err := WithTransaction(context.Background(), db, func(ctx context.Context, tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO Items (Name) VALUES (?)", "alpha")
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, 1, itemCount(t, db))
}

func TestWithTransaction_RollsBackOnError(t *testing.T) {
	db := openTxTestDB(t)
	want := errors.New("fn failed")

	err := WithTransaction(context.Background(), db, func(ctx context.Context, tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO Items (Name) VALUES (?)", "alpha"); err != nil {
			return err
		}
		return want
	})
	require.Error(t, err)
	require.ErrorIs(t, err, want)
	assert.Equal(t, 0, itemCount(t, db), "rolled back insert must not be visible after WithTransaction returns")
}

func TestWithTransaction_RollsBackOnPanic(t *testing.T) {
	db := openTxTestDB(t)

	require.Panics(t, func() {
		_ = WithTransaction(context.Background(), db, func(ctx context.Context, tx *sqlx.Tx) error {
			_, err := tx.ExecContext(ctx, "INSERT INTO Items (Name) VALUES (?)", "alpha")
			require.NoError(t, err)
			panic("boom")
		})
	})
	assert.Equal(t, 0, itemCount(t, db), "deferred rollback must run when fn panics")
}

func TestWithTransaction_HonorsCallerContext(t *testing.T) {
	db := openTxTestDB(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := WithTransaction(ctx, db, func(ctx context.Context, tx *sqlx.Tx) error {
		return nil
	})
	require.Error(t, err, "WithTransaction should propagate a cancelled caller context")
}
