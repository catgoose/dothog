// setup:feature:graph

package graph

import (
	"context"
	"fmt"

	"catgoose/dothog/internal/database"
	"catgoose/dothog/internal/database/schema"

	_ "github.com/catgoose/chuck/driver/sqlite" // Register SQLite driver
	"github.com/jmoiron/sqlx"
)

// OpenDirectory opens the Graph-owned, on-disk SQLite cache that backs both
// the user directory and the avatar photo blobs. The file survives restart
// so derived apps can serve the last snapshot immediately while the next
// sync runs. UsersTable and PhotosTable are materialized at open time;
// callers use the returned *Directory's narrow accessors instead of holding
// the raw DB handle.
func OpenDirectory(ctx context.Context, path string) (*Directory, error) {
	db, err := database.OpenSQLite(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("open graph directory db: %w", err)
	}
	if err := materializeCacheTables(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Directory{db: db}, nil
}

// materializeCacheTables ensures UsersTable and PhotosTable both exist in
// the Graph directory DB. OpenDirectory and Directory.EnsureSchema both
// route through this so the schema has exactly one source of truth.
func materializeCacheTables(ctx context.Context, db *sqlx.DB) error {
	d, err := database.SQLiteDialect()
	if err != nil {
		return fmt.Errorf("failed to create SQLite dialect: %w", err)
	}
	mat := schema.NewMaterializer(db, d, UsersTable, PhotosTable, MetaTable)
	return mat.EnsureSchema(ctx)
}
