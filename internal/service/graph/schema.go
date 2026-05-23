// setup:feature:graph

package graph

import "catgoose/dothog/internal/database/schema"

// UsersTable is the chuck-backed schema for the Graph directory cache. Lives
// in the same Graph-owned SQLite cache file as PhotosTable; both tables are
// materialized at open time by OpenDirectory.
var UsersTable = schema.NewTable("Users").
	Columns(
		schema.AutoIncrCol("ID"),
		schema.Col("AzureId", schema.TypeVarchar(255)).NotNull().Unique(),
		schema.Col("GivenName", schema.TypeString(255)),
		schema.Col("Surname", schema.TypeString(255)),
		schema.Col("DisplayName", schema.TypeString(255)),
		schema.Col("UserPrincipalName", schema.TypeString(255)).NotNull(),
		schema.Col("Mail", schema.TypeString(255)),
		schema.Col("JobTitle", schema.TypeString(255)),
		schema.Col("OfficeLocation", schema.TypeString(255)),
		schema.Col("Department", schema.TypeString(255)),
		schema.Col("CompanyName", schema.TypeString(255)),
		schema.Col("AccountName", schema.TypeString(255)),
	).
	WithTimestamps().
	Indexes(
		schema.Index("idx_users_azureid", "AzureId"),
		schema.Index("idx_users_userprincipalname", "UserPrincipalName"),
		schema.Index("idx_users_displayname", "DisplayName"),
		schema.Index("idx_users_mail", "Mail"),
	)

// PhotosTable holds cached avatar blobs alongside UsersTable. Rows include
// both fetched photos (non-null Bytes) and "checked and Graph has no photo"
// markers (null Bytes); LastCheckedAt drives the photo-sync freshness window
// so repeated startups don't re-query Graph for every user.
var PhotosTable = schema.NewTable("Photos").
	Columns(
		schema.Col("AzureId", schema.TypeVarchar(255)).NotNull().Unique(),
		schema.Col("ContentType", schema.TypeString(64)).NotNull().Default("'image/jpeg'"),
		schema.Col("Bytes", schema.TypeLiteral("BLOB")),
		schema.Col("FetchedAt", schema.TypeTimestamp()),
		schema.Col("LastCheckedAt", schema.TypeTimestamp()),
	).
	WithTimestamps().
	Indexes(
		schema.Index("idx_photos_azureid", "AzureId"),
	)

// MetaTable carries Graph-directory sync state: per-Kind timestamps for the
// last successful refresh. Kept minimal — one row per sync kind.
var MetaTable = schema.NewTable("DirectoryMeta").
	Columns(
		schema.Col("Kind", schema.TypeVarchar(64)).NotNull().Unique(),
		schema.Col("LastSuccessAt", schema.TypeTimestamp()),
	).
	WithTimestamps()
