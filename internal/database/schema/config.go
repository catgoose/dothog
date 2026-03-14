package schema

import (
	"fmt"
	"strings"
)

// NewConfigTable creates a key-value settings table with a UNIQUE key column and a TEXT value column.
// Caller provides the table name and column names for flexibility (e.g., "Key"/"Value" or "Name"/"Data").
func NewConfigTable(name, keyCol, valueCol string) *TableDef {
	lower := strings.ToLower(name)
	keyLower := strings.ToLower(keyCol)
	return NewTable(name).
		Columns(
			AutoIncrCol("ID"),
			Col(keyCol, TypeVarchar(255)).NotNull().Unique(),
			Col(valueCol, TypeText()),
		).
		Indexes(
			Index(fmt.Sprintf("idx_%s_%s", lower, keyLower), keyCol),
		)
}
