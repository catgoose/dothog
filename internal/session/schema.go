// setup:feature:session_settings

package session

import "catgoose/dothog/internal/database/schema"

// SettingsTable is the chuck-backed schema for the session_settings table.
// Session settings live in their own SQLite store (db/session_settings.db);
// this table def drives the schema.Materializer that ensures the table on
// startup and is shared by SessionSettingsRepository for column-name reuse.
var SettingsTable = schema.NewTable("SessionSettings").
	Columns(
		schema.AutoIncrCol("Id"),
		schema.Col("SessionUUID", schema.TypeVarchar(36)).NotNull().Unique(),
		schema.Col("Theme", schema.TypeString(50)).NotNull().Default("'light'"),
		schema.Col("Layout", schema.TypeString(50)).NotNull().Default("'classic'"),
	).
	WithTimestamps()
