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
	// SeedRow is a re-exported chuck/schema seed row.
	SeedRow = s.SeedRow
	// TypeFunc is a re-exported chuck/schema type constructor.
	TypeFunc = s.TypeFunc
)

// Re-export chuck/schema constructors.
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
