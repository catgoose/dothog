package schema

import "catgoose/go-htmx-demo/internals/database/dialect"

// TimestampColumnDefs returns CreatedAt and UpdatedAt column definitions.
func TimestampColumnDefs() []ColumnDef {
	return []ColumnDef{
		Col("CreatedAt", TypeTimestamp()).NotNull().
			DefaultFn(func(d dialect.Dialect) string { return d.Now() }).Immutable(),
		Col("UpdatedAt", TypeTimestamp()).NotNull().
			DefaultFn(func(d dialect.Dialect) string { return d.Now() }),
	}
}

// SoftDeleteColumnDefs returns a nullable DeletedAt timestamp column.
func SoftDeleteColumnDefs() []ColumnDef {
	return []ColumnDef{
		Col("DeletedAt", TypeTimestamp()),
	}
}

// AuditColumnDefs returns CreatedBy, UpdatedBy, and DeletedBy columns.
func AuditColumnDefs() []ColumnDef {
	return []ColumnDef{
		Col("CreatedBy", TypeString(255)).Immutable(),
		Col("UpdatedBy", TypeString(255)),
		Col("DeletedBy", TypeString(255)),
	}
}
