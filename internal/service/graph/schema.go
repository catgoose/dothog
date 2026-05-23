// setup:feature:graph

package graph

import "catgoose/dothog/internal/database/schema"

// UsersTable is the chuck-backed schema for the Graph user cache and the
// app-data Users repository. Both consumers — the in-memory cache opened by
// OpenUserCache and any chuck-backed repository that mirrors Graph users —
// derive their schema from this single definition so column shape and
// indexes stay in lockstep.
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
		schema.Col("LastLoginAt", schema.TypeTimestamp()),
	).
	WithTimestamps().
	Indexes(
		schema.Index("idx_users_azureid", "AzureId"),
		schema.Index("idx_users_userprincipalname", "UserPrincipalName"),
		schema.Index("idx_users_displayname", "DisplayName"),
		schema.Index("idx_users_mail", "Mail"),
		schema.Index("idx_users_lastloginat", "LastLoginAt"),
	)
