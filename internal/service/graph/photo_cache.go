// setup:feature:avatar

package graph

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/catgoose/chuck/dbrepo"
	"github.com/jmoiron/sqlx"
)

// PhotoCache reads/writes cached Graph avatar blobs in the Graph-owned
// SQLite cache (PhotosTable). Obtain one via *Directory.Photos(); the
// underlying DB pool is shared with the parent Directory. Rows track three
// states: photo present (Bytes set), checked-and-missing (Bytes NULL,
// LastCheckedAt set), and never-checked (no row).
type PhotoCache struct {
	db *sqlx.DB
}

// Has reports whether a cached row has fetched photo bytes for azureID.
// Returns false for checked-and-missing rows and for never-checked users.
func (p *PhotoCache) Has(ctx context.Context, azureID string) bool {
	var n int
	err := p.db.GetContext(ctx, &n,
		"SELECT 1 FROM "+PhotosTable.Name+" WHERE AzureId = @AzureId AND Bytes IS NOT NULL LIMIT 1",
		sql.Named("AzureId", azureID),
	)
	if err != nil {
		return false
	}
	return n == 1
}

// Get returns the cached photo bytes and content type for azureID. Returns
// sql.ErrNoRows when the row is missing OR when the row records a
// checked-and-missing entry (Bytes NULL), so callers can treat both as 404.
func (p *PhotoCache) Get(ctx context.Context, azureID string) (data []byte, contentType string, err error) {
	row := struct {
		ContentType string `db:"ContentType"`
		Bytes       []byte `db:"Bytes"`
	}{}
	if err := p.db.GetContext(ctx, &row,
		"SELECT Bytes, ContentType FROM "+PhotosTable.Name+" WHERE AzureId = @AzureId AND Bytes IS NOT NULL",
		sql.Named("AzureId", azureID),
	); err != nil {
		return nil, "", err
	}
	return row.Bytes, row.ContentType, nil
}

// LastChecked returns when this AzureId was last queried against Graph and
// whether a row was found. A false `found` means we've never queried; a true
// `found` means we have — independent of whether the response included a
// photo.
func (p *PhotoCache) LastChecked(ctx context.Context, azureID string) (time.Time, bool, error) {
	var t sql.NullTime
	err := p.db.GetContext(ctx, &t,
		"SELECT LastCheckedAt FROM "+PhotosTable.Name+" WHERE AzureId = @AzureId",
		sql.Named("AzureId", azureID),
	)
	if err == sql.ErrNoRows {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("last checked for %s: %w", azureID, err)
	}
	if !t.Valid {
		return time.Time{}, true, nil
	}
	return t.Time, true, nil
}

// Save upserts the photo for azureID. Empty data is rejected — callers that
// observe a Graph "no photo" response should use MarkMissing instead so the
// next sync can skip the user during the freshness window.
func (p *PhotoCache) Save(ctx context.Context, azureID, contentType string, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("refusing to cache empty photo bytes for %s (use MarkMissing for Graph 404 responses)", azureID)
	}
	if contentType == "" {
		contentType = "image/jpeg"
	}
	now := dbrepo.GetNow()
	_, err := p.db.ExecContext(ctx,
		"INSERT INTO "+PhotosTable.Name+" (AzureId, ContentType, Bytes, FetchedAt, LastCheckedAt, CreatedAt, UpdatedAt) "+
			"VALUES (@AzureId, @ContentType, @Bytes, @FetchedAt, @LastCheckedAt, @CreatedAt, @UpdatedAt) "+
			"ON CONFLICT(AzureId) DO UPDATE SET "+
			"ContentType = excluded.ContentType, "+
			"Bytes = excluded.Bytes, "+
			"FetchedAt = excluded.FetchedAt, "+
			"LastCheckedAt = excluded.LastCheckedAt, "+
			"UpdatedAt = excluded.UpdatedAt",
		sql.Named("AzureId", azureID),
		sql.Named("ContentType", contentType),
		sql.Named("Bytes", data),
		sql.Named("FetchedAt", now),
		sql.Named("LastCheckedAt", now),
		sql.Named("CreatedAt", now),
		sql.Named("UpdatedAt", now),
	)
	if err != nil {
		return fmt.Errorf("save photo for %s: %w", azureID, err)
	}
	return nil
}

// MarkMissing records that Graph reported no photo for azureID at the
// current moment. The row carries NULL Bytes; Has and Get treat it as 404,
// while LastChecked surfaces the timestamp so sync can skip the user until
// the freshness window expires.
func (p *PhotoCache) MarkMissing(ctx context.Context, azureID string) error {
	now := dbrepo.GetNow()
	_, err := p.db.ExecContext(ctx,
		"INSERT INTO "+PhotosTable.Name+" (AzureId, ContentType, Bytes, FetchedAt, LastCheckedAt, CreatedAt, UpdatedAt) "+
			"VALUES (@AzureId, @ContentType, NULL, NULL, @LastCheckedAt, @CreatedAt, @UpdatedAt) "+
			"ON CONFLICT(AzureId) DO UPDATE SET "+
			"Bytes = NULL, "+
			"FetchedAt = NULL, "+
			"LastCheckedAt = excluded.LastCheckedAt, "+
			"UpdatedAt = excluded.UpdatedAt",
		sql.Named("AzureId", azureID),
		sql.Named("ContentType", "image/jpeg"),
		sql.Named("LastCheckedAt", now),
		sql.Named("CreatedAt", now),
		sql.Named("UpdatedAt", now),
	)
	if err != nil {
		return fmt.Errorf("mark missing photo for %s: %w", azureID, err)
	}
	return nil
}
