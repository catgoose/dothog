package schema

import (
	"fmt"
	"strings"
)

// NewTagsTable creates a lookup-style tags table with ID and two caller-named columns.
// The groupCol categorizes tags (e.g., "Type", "Category") and the valueCol holds the
// display value (e.g., "Label", "Name"). Indexes are created on groupCol and (groupCol, valueCol).
func NewTagsTable(name, groupCol, valueCol string) *TableDef {
	lower := strings.ToLower(name)
	groupLower := strings.ToLower(groupCol)
	valueLower := strings.ToLower(valueCol)
	return NewTable(name).
		Columns(
			AutoIncrCol("ID"),
			Col(groupCol, TypeVarchar(100)).NotNull(),
			Col(valueCol, TypeVarchar(255)).NotNull(),
		).
		Indexes(
			Index(fmt.Sprintf("idx_%s_%s", lower, groupLower), groupCol),
			Index(fmt.Sprintf("idx_%s_%s_%s", lower, groupLower, valueLower), groupCol+", "+valueCol),
		)
}

// NewTagJoinTable creates a many-to-many join table linking an owner table to a tags table.
func NewTagJoinTable(name string) *TableDef {
	lower := strings.ToLower(name)
	return NewTable(name).
		Columns(
			Col("OwnerID", TypeInt()).NotNull().Immutable(),
			Col("TagID", TypeInt()).NotNull().Immutable(),
		).
		Indexes(
			Index(fmt.Sprintf("idx_%s_ownerid", lower), "OwnerID"),
			Index(fmt.Sprintf("idx_%s_tagid", lower), "TagID"),
		)
}
