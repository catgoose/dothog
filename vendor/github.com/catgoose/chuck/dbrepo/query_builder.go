package dbrepo

import (
	"fmt"
	"strings"
)

// BuildSearchPattern builds a LIKE search pattern from a search string.
// Returns "%search%" or empty string if search is empty.
func BuildSearchPattern(search string) string {
	if search == "" {
		return ""
	}
	return "%" + search + "%"
}

// BuildSearchCondition builds a search condition for WHERE clauses.
// Returns "(field1 LIKE @SearchPattern OR field2 LIKE @SearchPattern)".
func BuildSearchCondition(search, searchPattern string, fields ...string) string {
	if len(fields) == 0 || search == "" {
		return "1=1"
	}

	var conditions []string
	for _, field := range fields {
		conditions = append(conditions, fmt.Sprintf("%s LIKE @SearchPattern", field))
	}

	return "(" + strings.Join(conditions, " OR ") + ")"
}

// ParseSortString parses a sort string like "name:asc,created_at:desc" into column/direction pairs.
func ParseSortString(sortStr string) []SortField {
	if sortStr == "" {
		return nil
	}

	var fields []SortField
	for _, part := range strings.Split(sortStr, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pieces := strings.SplitN(part, ":", 2)
		col := strings.TrimSpace(pieces[0])
		dir := "ASC"
		if len(pieces) == 2 {
			d := strings.ToUpper(strings.TrimSpace(pieces[1]))
			if d == "DESC" {
				dir = "DESC"
			}
		}
		if col != "" {
			fields = append(fields, SortField{Column: col, Direction: dir})
		}
	}
	return fields
}

// SortField represents a column and sort direction.
type SortField struct {
	Column    string
	Direction string
}

// BuildOrderByClause builds an ORDER BY clause from a sort string.
// columnMap maps user-friendly column names to SQL column names.
// defaultSort is used when sortStr is empty or produces no valid columns.
func BuildOrderByClause(sortStr string, columnMap map[string]string, defaultSort string) string {
	if sortStr == "" {
		if defaultSort != "" {
			return "ORDER BY " + defaultSort
		}
		return ""
	}

	sorts := ParseSortString(sortStr)
	if len(sorts) == 0 {
		if defaultSort != "" {
			return "ORDER BY " + defaultSort
		}
		return ""
	}

	var orderParts []string
	for _, s := range sorts {
		sqlColumn, ok := columnMap[s.Column]
		if !ok {
			continue
		}
		orderParts = append(orderParts, fmt.Sprintf("%s %s", sqlColumn, s.Direction))
	}

	if len(orderParts) == 0 {
		if defaultSort != "" {
			return "ORDER BY " + defaultSort
		}
		return ""
	}

	return "ORDER BY " + strings.Join(orderParts, ", ")
}
