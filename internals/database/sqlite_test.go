// setup:feature:graph
package database

import (
	"context"
	"os"
	"testing"

	"catgoose/go-htmx-demo/internals/logger"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("LOG_LEVEL", "ERROR")
	logger.Init()
}

func TestOpenSQLiteInMemory(t *testing.T) {
	db, err := OpenSQLiteInMemory()
	require.NoError(t, err)
	require.NotNil(t, db)
	t.Cleanup(func() { _ = db.Close() })

	err = db.PingContext(context.Background())
	require.NoError(t, err)
}

func TestInitSQLiteUserCacheSchema(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	err = InitSQLiteUserCacheSchema(db)
	require.NoError(t, err)

	var count int
	err = db.GetContext(context.Background(), &count, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='Users'")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
