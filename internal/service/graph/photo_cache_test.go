// setup:feature:avatar
package graph

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openPhotoCacheForTest(t *testing.T) *PhotoCache {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "graph_cache.db")
	dir, err := OpenDirectory(ctx, dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = dir.Close() })
	return dir.Photos()
}

func TestPhotoCache_MarkMissing_RecordsCheckButReturnsErrNoRowsOnGet(t *testing.T) {
	ctx := context.Background()
	cache := openPhotoCacheForTest(t)

	require.NoError(t, cache.MarkMissing(ctx, "azure-x"))

	assert.False(t, cache.Has(ctx, "azure-x"), "MarkMissing must not count as having a photo")

	_, _, err := cache.Get(ctx, "azure-x")
	require.Error(t, err)
	assert.True(t, errors.Is(err, sql.ErrNoRows), "Get of a missing-marker row must surface as ErrNoRows for the route's 404 branch")

	_, found, err := cache.LastChecked(ctx, "azure-x")
	require.NoError(t, err)
	assert.True(t, found, "MarkMissing must set LastCheckedAt so sync can skip the user within the freshness window")
}

func TestPhotoCache_SaveAfterMarkMissingFlipsToBytes(t *testing.T) {
	ctx := context.Background()
	cache := openPhotoCacheForTest(t)

	require.NoError(t, cache.MarkMissing(ctx, "azure-y"))
	require.NoError(t, cache.Save(ctx, "azure-y", "image/png", []byte{0x89, 0x50, 0x4e, 0x47}))

	assert.True(t, cache.Has(ctx, "azure-y"), "Save after MarkMissing must flip the row back to having bytes")
	data, ct, err := cache.Get(ctx, "azure-y")
	require.NoError(t, err)
	assert.Equal(t, "image/png", ct)
	assert.Equal(t, []byte{0x89, 0x50, 0x4e, 0x47}, data)
}

func TestPhotoCache_LastChecked_AbsentRow(t *testing.T) {
	ctx := context.Background()
	cache := openPhotoCacheForTest(t)

	_, found, err := cache.LastChecked(ctx, "azure-never")
	require.NoError(t, err)
	assert.False(t, found, "never-checked azureID must surface as found=false")
}
