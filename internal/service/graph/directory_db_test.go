// setup:feature:graph
package graph

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenDirectory_MaterializesUsersAndPhotos(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "graph_cache.db")
	dir, err := OpenDirectory(ctx, dbPath)
	require.NoError(t, err)
	require.NotNil(t, dir)
	t.Cleanup(func() { _ = dir.Close() })

	// Same-package access lets the test probe the otherwise-hidden DB to
	// confirm the schema, without exposing the handle in production API.
	require.NoError(t, dir.db.PingContext(ctx))

	var usersCount, photosCount int
	require.NoError(t, dir.db.GetContext(ctx, &usersCount, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='Users'"))
	require.NoError(t, dir.db.GetContext(ctx, &photosCount, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='Photos'"))
	assert.Equal(t, 1, usersCount, "OpenDirectory must materialize the Users table")
	assert.Equal(t, 1, photosCount, "OpenDirectory must materialize the Photos table")
}

func TestOpenDirectory_SurvivesRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "graph_cache.db")

	dir1, err := OpenDirectory(ctx, dbPath)
	require.NoError(t, err)
	require.NoError(t, dir1.Photos().Save(ctx, "azure-1", "image/jpeg", []byte("bytes-v1")))
	require.NoError(t, dir1.Close())

	dir2, err := OpenDirectory(ctx, dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = dir2.Close() })
	photos := dir2.Photos()
	assert.True(t, photos.Has(ctx, "azure-1"), "photo cached before close must survive a reopen of the same file")
	data, ct, err := photos.Get(ctx, "azure-1")
	require.NoError(t, err)
	assert.Equal(t, "image/jpeg", ct)
	assert.Equal(t, []byte("bytes-v1"), data)
}
