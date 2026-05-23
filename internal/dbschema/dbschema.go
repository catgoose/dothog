// setup:feature:database

// Package dbschema is the starter seam for app-owned database tables in
// dothog-derived applications. Tables registered here flow into the
// chuck-backed schema.Materializer constructed in main.go, so derived apps can
// declare their schema once and have Init/Ensure/Validate apply it on
// startup. Framework-owned tables (session settings, graph user cache,
// error traces) live in their own packages and are NOT registered here.
//
// The package ships only with derived apps that keep the internal `database`
// feature — selected indirectly by picking MSSQL or PostgreSQL during setup.
// Scaffolds without app-data support never see this package.
//
// Adding a table is two steps:
//
//  1. Define the table next to its repository (e.g. internal/orders/schema.go):
//
//     var OrdersTable = schema.NewTable("Orders").
//         Columns(
//             schema.AutoIncrCol("Id"),
//             schema.Col("CustomerId", schema.TypeInt()).NotNull(),
//             schema.Col("Total", schema.TypeInt()).NotNull(),
//         ).
//         WithTimestamps()
//
//  2. Append it to Tables() below. The runtime picks it up automatically.
//
// Concrete repositories hold *sqlx.DB directly (no manager indirection) and
// run statements via GetContext / ExecContext / SelectContext. Multi-statement
// work uses database.WithTransaction(ctx, db, fn). The chuck dbrepo package
// provides query-builder helpers (NewSelect, NewWhere, SetClause, InsertInto).
// See internal/repository/session_settings_repository.go for a working
// framework-owned example of the pattern.
package dbschema

import "catgoose/dothog/internal/database/schema"

// Tables enumerates the app-owned schema tables. The template ships an empty
// list on purpose — derived apps extend it.
func Tables() []*schema.TableDef {
	return nil
}
