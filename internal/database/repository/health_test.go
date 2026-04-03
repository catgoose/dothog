// setup:feature:database
package repository

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/catgoose/chuck/driver/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openSQLiteInMemory(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestHealthCheck_Success(t *testing.T) {
	db := openSQLiteInMemory(t)
	ctx := context.Background()
	err := HealthCheck(ctx, db)
	require.NoError(t, err)
}

func TestHealthCheck_ClosedDB(t *testing.T) {
	db := openSQLiteInMemory(t)
	require.NoError(t, db.Close())
	ctx := context.Background()
	err := HealthCheck(ctx, db)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "health check failed")
}

func TestCheckConnection_Success(t *testing.T) {
	db := openSQLiteInMemory(t)
	ctx := context.Background()
	ok, err := CheckConnection(ctx, db)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCheckConnection_ClosedDB(t *testing.T) {
	db := openSQLiteInMemory(t)
	require.NoError(t, db.Close())
	ctx := context.Background()
	ok, err := CheckConnection(ctx, db)
	require.Error(t, err)
	assert.False(t, ok)
	assert.Contains(t, err.Error(), "connection check failed")
}
