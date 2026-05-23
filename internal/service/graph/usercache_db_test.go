// setup:feature:graph
package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenUserCacheDB(t *testing.T) {
	db, err := OpenUserCacheDB()
	require.NoError(t, err)
	require.NotNil(t, db)
	t.Cleanup(func() { _ = db.Close() })

	err = db.PingContext(context.Background())
	require.NoError(t, err)

	var count int
	err = db.GetContext(context.Background(), &count, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='Users'")
	require.NoError(t, err)
	assert.Equal(t, 1, count, "OpenUserCacheDB must materialize the Users table at open time")
}
