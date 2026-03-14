// setup:feature:demo
package demo

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedDBPath(t *testing.T) string {
	t.Helper()
	// Look for seed.db relative to project root; return absolute path
	// so SQLite ATTACH resolves correctly regardless of working directory.
	for _, p := range []string{"db/seed.db", "../../db/seed.db"} {
		if _, err := os.Stat(p); err == nil {
			abs, err := filepath.Abs(p)
			require.NoError(t, err)
			return abs
		}
	}
	t.Skip("seed.db not found")
	return ""
}

func openTestMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := OpenMemoryDB()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSeedDB_AttachDetach(t *testing.T) {
	path := seedDBPath(t)
	db := openTestMemoryDB(t)
	s := NewSeedDB(db)
	ctx := context.Background()

	assert.False(t, s.IsAttached())

	require.NoError(t, s.Attach(ctx, path))
	assert.True(t, s.IsAttached())

	// Double attach is safe
	require.NoError(t, s.Attach(ctx, path))

	require.NoError(t, s.Detach(ctx))
	assert.False(t, s.IsAttached())
}

func TestSeedDB_SeedTables(t *testing.T) {
	path := seedDBPath(t)
	db := openTestMemoryDB(t)
	s := NewSeedDB(db)
	ctx := context.Background()

	require.NoError(t, s.Attach(ctx, path))
	tables, err := s.SeedTables(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, tables)

	tableNames := make([]string, len(tables))
	for i, tbl := range tables {
		tableNames[i] = tbl.Name
		assert.Greater(t, tbl.RowCount, 0, "table %s should have rows", tbl.Name)
	}
	assert.Contains(t, tableNames, "first_names")
	assert.Contains(t, tableNames, "last_names")
	assert.Contains(t, tableNames, "states")
	assert.Contains(t, tableNames, "cities")
	assert.Contains(t, tableNames, "vendors")
}

func TestSeedDB_CopyTable(t *testing.T) {
	path := seedDBPath(t)
	db := openTestMemoryDB(t)
	s := NewSeedDB(db)
	ctx := context.Background()

	require.NoError(t, s.Attach(ctx, path))

	rows, err := s.CopyTable(ctx, "states")
	require.NoError(t, err)
	assert.Equal(t, int64(50), rows)

	// Verify in main DB
	mainTables, err := s.MainTables(ctx)
	require.NoError(t, err)
	found := false
	for _, tbl := range mainTables {
		if tbl.Name == "states" {
			found = true
			assert.Equal(t, 50, tbl.RowCount)
		}
	}
	assert.True(t, found, "states table should exist in main DB")
}

func TestSeedDB_DropMainTable(t *testing.T) {
	path := seedDBPath(t)
	db := openTestMemoryDB(t)
	s := NewSeedDB(db)
	ctx := context.Background()

	require.NoError(t, s.Attach(ctx, path))
	_, err := s.CopyTable(ctx, "vendors")
	require.NoError(t, err)

	require.NoError(t, s.DropMainTable(ctx, "vendors"))

	mainTables, err := s.MainTables(ctx)
	require.NoError(t, err)
	for _, tbl := range mainTables {
		assert.NotEqual(t, "vendors", tbl.Name)
	}
}

func TestSeedDB_NotAttached(t *testing.T) {
	db := openTestMemoryDB(t)
	s := NewSeedDB(db)
	ctx := context.Background()

	_, err := s.SeedTables(ctx)
	assert.Error(t, err)

	_, err = s.CopyTable(ctx, "states")
	assert.Error(t, err)
}
