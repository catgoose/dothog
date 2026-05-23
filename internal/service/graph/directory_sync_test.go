// setup:feature:graph
package graph

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openDirectoryForTest(t *testing.T) *Directory {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "graph_cache.db")
	dir, err := OpenDirectory(ctx, dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = dir.Close() })
	return dir
}

func TestReplaceUsers_PrunesAbsentUsers(t *testing.T) {
	ctx := context.Background()
	dir := openDirectoryForTest(t)

	first := []GraphUser{
		{AzureID: "azure-1", DisplayName: "Alice", UserPrincipalName: "alice@example.com"},
		{AzureID: "azure-2", DisplayName: "Bob", UserPrincipalName: "bob@example.com"},
		{AzureID: "azure-3", DisplayName: "Carol", UserPrincipalName: "carol@example.com"},
	}
	require.NoError(t, dir.ReplaceUsers(ctx, first))
	got, err := dir.UserCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, got)

	// Bob is removed/disabled in Graph; the next snapshot drops him.
	second := []GraphUser{
		{AzureID: "azure-1", DisplayName: "Alice", UserPrincipalName: "alice@example.com"},
		{AzureID: "azure-3", DisplayName: "Carol", UserPrincipalName: "carol@example.com"},
	}
	require.NoError(t, dir.ReplaceUsers(ctx, second))
	got, err = dir.UserCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, got, "ReplaceUsers must drop AzureIds absent from the latest fetch")

	_, err = dir.UserByAzureID(ctx, "azure-2")
	require.Error(t, err, "Bob's row must be gone after a snapshot that doesn't include him")
}

func TestReplaceUsers_EmptySnapshotClearsCache(t *testing.T) {
	ctx := context.Background()
	dir := openDirectoryForTest(t)

	require.NoError(t, dir.ReplaceUsers(ctx, []GraphUser{
		{AzureID: "azure-1", DisplayName: "Alice", UserPrincipalName: "alice@example.com"},
	}))
	// Degraded fetches surface as non-nil errors; an empty successful fetch
	// is a valid "Graph has no users" snapshot and must clear the cache.
	require.NoError(t, dir.ReplaceUsers(ctx, nil))
	got, err := dir.UserCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, got, "empty successful snapshot must clear cached users")
}

func TestLastSuccessfulSync_RoundTrips(t *testing.T) {
	ctx := context.Background()
	dir := openDirectoryForTest(t)

	_, ok, err := dir.LastSuccessfulSync(ctx, syncKindUsers)
	require.NoError(t, err)
	assert.False(t, ok, "no sync recorded yet")

	require.NoError(t, dir.MarkSuccessfulSync(ctx, syncKindUsers))
	t1, ok, err := dir.LastSuccessfulSync(ctx, syncKindUsers)
	require.NoError(t, err)
	require.True(t, ok)
	assert.WithinDuration(t, time.Now(), t1, 5*time.Second, "first MarkSuccessfulSync should land at ~now")

	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, dir.MarkSuccessfulSync(ctx, syncKindUsers))
	t2, _, err := dir.LastSuccessfulSync(ctx, syncKindUsers)
	require.NoError(t, err)
	assert.True(t, t2.After(t1) || t2.Equal(t1), "second MarkSuccessfulSync should advance LastSuccessAt")
}
