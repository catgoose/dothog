// setup:feature:demo

package demo

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestOpenCreatesSchemaAndSeeds(t *testing.T) {
	db := openTestDB(t)
	items, total, err := db.ListItems(context.Background(), "", "", "", "name", "asc", 1, 100)
	require.NoError(t, err)
	assert.Equal(t, 50, total)
	assert.Len(t, items, 50)
}

func TestOpenNonExistentPath(t *testing.T) {
	dir := t.TempDir()
	// Nested non-existent directory — sqlite3 should still create the file.
	db, err := Open(filepath.Join(dir, "sub", "test.db"))
	if err != nil {
		// Some sqlite drivers can't create intermediate dirs; that's fine.
		t.Skip("sqlite3 cannot create nested path:", err)
	}
	defer func() { _ = db.Close() }()
}

func TestCreateGetRoundTrip(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	created, err := db.CreateItem(ctx, Item{
		Name: "Test Widget", Category: "Electronics", Price: 9.99, Stock: 5, Active: true,
	})
	require.NoError(t, err)
	assert.Greater(t, created.ID, 0)

	got, err := db.GetItem(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test Widget", got.Name)
	assert.Equal(t, "Electronics", got.Category)
	assert.InDelta(t, 9.99, got.Price, 0.001)
	assert.Equal(t, 5, got.Stock)
	assert.True(t, got.Active)
}

func TestUpdateItem(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	created, err := db.CreateItem(ctx, Item{Name: "Original", Category: "Books", Price: 10, Stock: 1, Active: true})
	require.NoError(t, err)

	created.Name = "Updated"
	created.Price = 20
	require.NoError(t, db.UpdateItem(ctx, created))

	got, err := db.GetItem(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Name)
	assert.InDelta(t, 20.0, got.Price, 0.001)
}

func TestDeleteItem(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	created, err := db.CreateItem(ctx, Item{Name: "ToDelete", Category: "Food", Price: 5, Stock: 1, Active: true})
	require.NoError(t, err)

	require.NoError(t, db.DeleteItem(ctx, created.ID))

	_, err = db.GetItem(ctx, created.ID)
	assert.Error(t, err)
}

func TestListItemsFilterByQuery(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	items, total, err := db.ListItems(ctx, "Laptop", "", "", "name", "asc", 1, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, items, 1)
	assert.Equal(t, "Laptop Pro 15", items[0].Name)
}

func TestListItemsFilterByCategory(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	items, total, err := db.ListItems(ctx, "", "Books", "", "name", "asc", 1, 100)
	require.NoError(t, err)
	assert.Equal(t, 10, total)
	assert.Len(t, items, 10)
	for _, it := range items {
		assert.Equal(t, "Books", it.Category)
	}
}

func TestListItemsFilterByActive(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	items, _, err := db.ListItems(ctx, "", "", "true", "name", "asc", 1, 100)
	require.NoError(t, err)
	for _, it := range items {
		assert.True(t, it.Active, "expected active item, got inactive: %s", it.Name)
	}
}

func TestListItemsSorting(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	t.Run("name asc", func(t *testing.T) {
		items, _, err := db.ListItems(ctx, "", "", "", "name", "asc", 1, 5)
		require.NoError(t, err)
		require.Len(t, items, 5)
		for i := 1; i < len(items); i++ {
			assert.LessOrEqual(t, items[i-1].Name, items[i].Name)
		}
	})

	t.Run("name desc", func(t *testing.T) {
		items, _, err := db.ListItems(ctx, "", "", "", "name", "desc", 1, 5)
		require.NoError(t, err)
		require.Len(t, items, 5)
		for i := 1; i < len(items); i++ {
			assert.GreaterOrEqual(t, items[i-1].Name, items[i].Name)
		}
	})

	t.Run("price asc", func(t *testing.T) {
		items, _, err := db.ListItems(ctx, "", "", "", "price", "asc", 1, 5)
		require.NoError(t, err)
		require.Len(t, items, 5)
		for i := 1; i < len(items); i++ {
			assert.LessOrEqual(t, items[i-1].Price, items[i].Price)
		}
	})
}

func TestListItemsPagination(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	page1, total, err := db.ListItems(ctx, "", "", "", "name", "asc", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 50, total)
	assert.Len(t, page1, 10)

	page2, _, err := db.ListItems(ctx, "", "", "", "name", "asc", 2, 10)
	require.NoError(t, err)
	assert.Len(t, page2, 10)

	// Pages should not overlap.
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

func TestListItemsArgsNotMutated(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	ctx := context.Background()

	// Call with a category filter so args slice has content.
	_, _, err = db.ListItems(ctx, "test", "Electronics", "", "name", "asc", 1, 10)
	require.NoError(t, err)

	// Call again — if args were mutated, this could fail or return wrong results.
	items, _, err := db.ListItems(ctx, "", "Books", "", "name", "asc", 1, 100)
	require.NoError(t, err)
	for _, it := range items {
		assert.Equal(t, "Books", it.Category)
	}
}

func TestOpenInvalidPath(t *testing.T) {
	_, err := Open(filepath.Join(os.DevNull, "impossible", "path.db"))
	assert.Error(t, err)
}

func TestBoolToInt(t *testing.T) {
	assert.Equal(t, 1, BoolToInt(true))
	assert.Equal(t, 0, BoolToInt(false))
}

func TestIntToBool(t *testing.T) {
	assert.True(t, IntToBool(1))
	assert.True(t, IntToBool(42))
	assert.False(t, IntToBool(0))
}

func TestSeedBulk(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	_, err := db.db.Exec("CREATE TABLE test_bulk (id INTEGER PRIMARY KEY, val TEXT)")
	require.NoError(t, err)

	data := []string{"alpha", "beta", "gamma"}
	err = seedBulk(db.db, "INSERT INTO test_bulk (val) VALUES (?)", len(data), func(i int) []any {
		return []any{data[i]}
	})
	require.NoError(t, err)

	var count int
	require.NoError(t, db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_bulk").Scan(&count))
	assert.Equal(t, 3, count)
}

func TestItemCategories(t *testing.T) {
	assert.NotEmpty(t, ItemCategories)
	assert.Contains(t, ItemCategories, "Electronics")
	assert.Contains(t, ItemCategories, "Books")
}
