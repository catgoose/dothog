package schema

import (
	s "github.com/catgoose/fraggle/schema"
)

// Re-export fraggle/schema types so consumers can continue to import
// this package for both table definitions and schema types.
type (
	TableDef  = s.TableDef
	ColumnDef = s.ColumnDef
	IndexDef  = s.IndexDef
	SeedRow   = s.SeedRow
	TypeFunc  = s.TypeFunc
)

// Re-export fraggle/schema constructors.
var (
	NewTable           = s.NewTable
	Col                = s.Col
	AutoIncrCol        = s.AutoIncrCol
	Index              = s.Index
	NewLookupTable     = s.NewLookupTable
	NewLookupJoinTable = s.NewLookupJoinTable
	NewMappingTable    = s.NewMappingTable
	NewConfigTable     = s.NewConfigTable
	NewEventTable      = s.NewEventTable
	NewQueueTable      = s.NewQueueTable
)

// Re-export type functions.
var (
	TypeInt       = s.TypeInt
	TypeText      = s.TypeText
	TypeString    = s.TypeString
	TypeVarchar   = s.TypeVarchar
	TypeTimestamp = s.TypeTimestamp
	TypeLiteral   = s.TypeLiteral
)

// setup:feature:session_settings:start

var SessionSettingsTable = NewTable("SessionSettings").
	Columns(
		AutoIncrCol("Id"),
		Col("SessionUUID", TypeVarchar(36)).NotNull().Unique(),
		Col("Theme", TypeString(50)).NotNull().Default("'light'"),
		Col("Layout", TypeString(50)).NotNull().Default("'classic'"),
	).
	WithTimestamps()

// setup:feature:session_settings:end

// setup:feature:graph:start

var UsersTable = NewTable("Users").
	Columns(
		AutoIncrCol("ID"),
		Col("AzureId", TypeVarchar(255)).NotNull().Unique(),
		Col("GivenName", TypeString(255)),
		Col("Surname", TypeString(255)),
		Col("DisplayName", TypeString(255)),
		Col("UserPrincipalName", TypeString(255)).NotNull(),
		Col("Mail", TypeString(255)),
		Col("JobTitle", TypeString(255)),
		Col("OfficeLocation", TypeString(255)),
		Col("Department", TypeString(255)),
		Col("CompanyName", TypeString(255)),
		Col("AccountName", TypeString(255)),
		Col("LastLoginAt", TypeTimestamp()),
	).
	WithTimestamps().
	Indexes(
		Index("idx_users_azureid", "AzureId"),
		Index("idx_users_userprincipalname", "UserPrincipalName"),
		Index("idx_users_displayname", "DisplayName"),
		Index("idx_users_mail", "Mail"),
		Index("idx_users_lastloginat", "LastLoginAt"),
	)

// setup:feature:graph:end
