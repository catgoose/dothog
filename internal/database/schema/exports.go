// Package schema re-exports the chuck/schema DSL surface for derived apps
// (table builders, column constructors, type functions) and bundles the
// generic schema.Materializer that runs InitSchema/EnsureSchema/SeedSchema/
// ValidateSchema against a *sqlx.DB. Feature-owned table definitions live in
// the packages that actually own them (e.g. internal/session for session
// settings, internal/service/graph for the Graph user cache).
package schema

import (
	s "github.com/catgoose/chuck/schema"
)

type (
	// TableDef is a re-exported chuck/schema table definition.
	TableDef = s.TableDef
	// ColumnDef is a re-exported chuck/schema column definition.
	ColumnDef = s.ColumnDef
	// IndexDef is a re-exported chuck/schema index definition.
	IndexDef = s.IndexDef
	// AuditActorsSpec configures CreatedBy / UpdatedBy audit actor columns.
	AuditActorsSpec = s.AuditActorsSpec
	// RetiredTableDef is an explicit tombstone for retired managed tables.
	RetiredTableDef = s.RetiredTableDef
	// SeedRow is a re-exported chuck/schema seed row.
	SeedRow = s.SeedRow
	// SeedValues is a re-exported typed seed row.
	SeedValues = s.SeedValues
	// TypeFunc is a re-exported chuck/schema type constructor.
	TypeFunc = s.TypeFunc
)

// Re-export chuck/schema constructors.
var (
	NewTable              = s.NewTable
	NewQualifiedTable     = s.NewQualifiedTable
	Col                   = s.Col
	AutoIncrCol           = s.AutoIncrCol
	UUIDPKCol             = s.UUIDPKCol
	Index                 = s.Index
	UniqueIndex           = s.UniqueIndex
	PartialIndex          = s.PartialIndex
	UniquePartialIndex    = s.UniquePartialIndex
	NewLookupTable        = s.NewLookupTable
	NewLookupJoinTable    = s.NewLookupJoinTable
	NewMappingTable       = s.NewMappingTable
	NewConfigTable        = s.NewConfigTable
	NewEventTable         = s.NewEventTable
	NewQueueTable         = s.NewQueueTable
	RetiredTable          = s.RetiredTable
	RetiredQualifiedTable = s.RetiredQualifiedTable
)

// Re-export type functions.
var (
	TypeInt           = s.TypeInt
	TypeBigInt        = s.TypeBigInt
	TypeFloat         = s.TypeFloat
	TypeDecimal       = s.TypeDecimal
	TypeText          = s.TypeText
	TypeString        = s.TypeString
	TypeVarchar       = s.TypeVarchar
	TypeTimestamp     = s.TypeTimestamp
	TypeAutoIncrement = s.TypeAutoIncrement
	TypeBool          = s.TypeBool
	TypeUUID          = s.TypeUUID
	TypeUUIDPK        = s.TypeUUIDPK
	TypeJSON          = s.TypeJSON
	TypeLiteral       = s.TypeLiteral
)

// Re-export trait helpers and presets.
var (
	TimestampColumnDefs      = s.TimestampColumnDefs
	SoftDeleteColumnDefs     = s.SoftDeleteColumnDefs
	AuditActorColumnDefs     = s.AuditActorColumnDefs
	DeleteActorColumnDef     = s.DeleteActorColumnDef
	DefaultStringAuditActors = s.DefaultStringAuditActors
	DefaultStringDeleteActor = s.DefaultStringDeleteActor
	VersionColumnDefs        = s.VersionColumnDefs
	SortOrderColumnDefs      = s.SortOrderColumnDefs
	StatusColumnDefs         = s.StatusColumnDefs
	NotesColumnDefs          = s.NotesColumnDefs
	UUIDColumnDefs           = s.UUIDColumnDefs
	ParentColumnDefs         = s.ParentColumnDefs
	ExpiryColumnDefs         = s.ExpiryColumnDefs
	ReplacementColumnDefs    = s.ReplacementColumnDefs
	ArchiveColumnDefs        = s.ArchiveColumnDefs
)

// FromStruct builds a TableDef from struct field tags.
func FromStruct[T any](name string) *TableDef {
	return s.FromStruct[T](name)
}
