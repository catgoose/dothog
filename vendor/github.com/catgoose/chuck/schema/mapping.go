package schema

import (
	"fmt"
	"strings"
)

// NewMappingTable creates a generic many-to-many join table with caller-defined column names.
// Both columns are NOT NULL, immutable, and indexed. A composite unique constraint is added
// to prevent duplicate associations.
func NewMappingTable(name, leftCol, rightCol string) *TableDef {
	lower := strings.ToLower(name)
	leftLower := strings.ToLower(leftCol)
	rightLower := strings.ToLower(rightCol)
	return NewTable(name).
		Columns(
			Col(leftCol, TypeInt()).NotNull().Immutable(),
			Col(rightCol, TypeInt()).NotNull().Immutable(),
		).
		UniqueColumns(leftCol, rightCol).
		Indexes(
			Index(fmt.Sprintf("idx_%s_%s", lower, leftLower), leftCol),
			Index(fmt.Sprintf("idx_%s_%s", lower, rightLower), rightCol),
		)
}
