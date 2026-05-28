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

	first := []User{
		{AzureID: "azure-1", DisplayName: "Alice", UserPrincipalName: "alice@example.com"},
		{AzureID: "azure-2", DisplayName: "Bob", UserPrincipalName: "bob@example.com"},
		{AzureID: "azure-3", DisplayName: "Carol", UserPrincipalName: "carol@example.com"},
	}
	require.NoError(t, dir.ReplaceUsers(ctx, first))
	got, err := dir.UserCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, got)

	// Bob is removed/disabled in Graph; the next snapshot drops him.
	second := []User{
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

	require.NoError(t, dir.ReplaceUsers(ctx, []User{
		{AzureID: "azure-1", DisplayName: "Alice", UserPrincipalName: "alice@example.com"},
	}))
	// Degraded fetches surface as non-nil errors; an empty successful fetch
	// is a valid "Graph has no users" snapshot and must clear the cache.
	require.NoError(t, dir.ReplaceUsers(ctx, nil))
	got, err := dir.UserCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, got, "empty successful snapshot must clear cached users")
}

// TestSearchUsers_TrustsCacheRowsWithoutFieldEligibility proves SearchUsers
// carries no SQL-side eligibility predicate on
// Mail/JobTitle/Department/CompanyName/AzureId. Rows with those fields empty
// must still match a name-style term, because eligibility is enforced
// upstream at Graph ingest, not at cache read time.
func TestSearchUsers_TrustsCacheRowsWithoutFieldEligibility(t *testing.T) {
	ctx := context.Background()
	dir := openDirectoryForTest(t)

	users := []User{
		{
			AzureID:           "azure-empty-fields",
			GivenName:         "Ada",
			Surname:           "Lovelace",
			DisplayName:       "Ada Lovelace",
			UserPrincipalName: "ada@example.com",
			AccountName:       "ada",
			// Mail, JobTitle, OfficeLocation, Department, CompanyName all empty.
		},
		{
			AzureID:     "azure-mail-only",
			DisplayName: "Grace Hopper",
			AccountName: "grace",
			Mail:        "grace@example.com",
		},
	}
	require.NoError(t, dir.ReplaceUsers(ctx, users))

	got, err := dir.SearchUsers(ctx, []string{"ada"}, 10)
	require.NoError(t, err)
	require.Len(t, got, 1, "row with empty Mail/JobTitle/etc. must still match by name term")
	assert.Equal(t, "azure-empty-fields", got[0].AzureID)
	assert.Empty(t, got[0].Mail, "cache row is returned as-is; no eligibility filter strips empty Mail")
	assert.Empty(t, got[0].JobTitle)

	got, err = dir.SearchUsers(ctx, []string{"grace"}, 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "azure-mail-only", got[0].AzureID)
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
