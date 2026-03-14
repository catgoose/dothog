// setup:feature:database

package repository

import (
	"fmt"
	"strings"

	"catgoose/dothog/internal/database/dialect"
	"catgoose/dothog/internal/routes/params"
)

// BuildPaginationClause builds the pagination clause for the given dialect.
//   - MSSQL:  "OFFSET @Offset ROWS FETCH NEXT @Limit ROWS ONLY"
//   - SQLite: "LIMIT @Limit OFFSET @Offset"
func BuildPaginationClause(d dialect.Dialect) string {
	return d.Pagination()
}

// BuildSearchPattern builds a search pattern for LIKE queries
// Returns: "%search%" pattern
func BuildSearchPattern(search string) string {
	if search == "" {
		return ""
	}
	return "%" + search + "%"
}

// BuildSearchCondition builds a search condition for WHERE clauses
// Returns: “(field1 LIKE @SearchPattern OR field2 LIKE @SearchPattern)”
func BuildSearchCondition(search, searchPattern string, fields ...string) string {
	if len(fields) == 0 {
		return "1=1" // No fields to search, return always true condition
	}

	if search == "" {
		return "1=1" // No search term, return always true condition
	}

	var conditions []string
	for _, field := range fields {
		conditions = append(conditions, fmt.Sprintf("%s LIKE @SearchPattern", field))
	}

	return "(" + strings.Join(conditions, " OR ") + ")"
}

// BuildOrderByClause builds an ORDER BY clause from a sort string
// columnMap maps user-friendly column names to SQL column names
// defaultSort is used when sortStr is empty or invalid
func BuildOrderByClause(sortStr string, columnMap map[string]string, defaultSort string) string {
	if sortStr == "" {
		if defaultSort != "" {
			return "ORDER BY " + defaultSort
		}
		return ""
	}

	sorts := params.ParseSortString(sortStr)
	if len(sorts) == 0 {
		if defaultSort != "" {
			return "ORDER BY " + defaultSort
		}
		return ""
	}

	var orderParts []string
	for _, sort := range sorts {
		// Map user-friendly column name to SQL column name
		sqlColumn, ok := columnMap[sort.Column]
		if !ok {
			continue // Skip unknown columns
		}

		// Validate direction
		direction := strings.ToUpper(sort.Direction)
		if direction != "ASC" && direction != "DESC" {
			direction = "ASC" // Default to ASC if invalid
		}

		orderParts = append(orderParts, fmt.Sprintf("%s %s", sqlColumn, direction))
	}

	if len(orderParts) == 0 {
		if defaultSort != "" {
			return "ORDER BY " + defaultSort
		}
		return ""
	}

	return "ORDER BY " + strings.Join(orderParts, ", ")
}
