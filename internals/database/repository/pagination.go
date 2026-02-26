// setup:feature:database

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"catgoose/go-htmx-template/internals/routes/params"

	"github.com/jmoiron/sqlx"
)

// PaginationResult holds paginated data and metadata
type PaginationResult[T any] struct {
	Data       []T
	TotalCount int
	Page       int
	Limit      int
	TotalPages int
}

// BuildCountQuery converts a SELECT query to a COUNT query
// It extracts the FROM clause and any WHERE conditions, then wraps them in COUNT(*)
func BuildCountQuery(selectQuery string) string {
	// Remove leading/trailing whitespace
	query := strings.TrimSpace(selectQuery)

	// Find the FROM clause (case-insensitive)
	fromIdx := -1
	upperQuery := strings.ToUpper(query)
	for i := 0; i < len(upperQuery)-4; i++ {
		if upperQuery[i:i+4] == "FROM" {
			// Check if it's a word boundary
			if i == 0 || (upperQuery[i-1] == ' ' || upperQuery[i-1] == '\n' || upperQuery[i-1] == '\t') {
				fromIdx = i
				break
			}
		}
	}

	if fromIdx == -1 {
		// No FROM clause found, return original query wrapped in COUNT
		return fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS count_query", query)
	}

	// Extract everything from FROM onwards
	fromClause := query[fromIdx:]

	// Remove ORDER BY, OFFSET, FETCH clauses for count query
	// Find ORDER BY (case-insensitive)
	orderByIdx := -1
	upperFromClause := strings.ToUpper(fromClause)
	for i := 0; i < len(upperFromClause)-8; i++ {
		if upperFromClause[i:i+8] == "ORDER BY" {
			// Check if it's a word boundary
			if i == 0 || (upperFromClause[i-1] == ' ' || upperFromClause[i-1] == '\n' || upperFromClause[i-1] == '\t') {
				orderByIdx = i
				break
			}
		}
	}

	if orderByIdx != -1 {
		fromClause = fromClause[:orderByIdx]
	}

	// Remove OFFSET/FETCH if present
	offsetIdx := -1
	upperFromClause = strings.ToUpper(fromClause)
	for i := 0; i < len(upperFromClause)-6; i++ {
		if upperFromClause[i:i+6] == "OFFSET" {
			// Check if it's a word boundary
			if i == 0 || (upperFromClause[i-1] == ' ' || upperFromClause[i-1] == '\n' || upperFromClause[i-1] == '\t') {
				offsetIdx = i
				break
			}
		}
	}

	if offsetIdx != -1 {
		fromClause = fromClause[:offsetIdx]
	}

	// Build count query
	countQuery := "SELECT COUNT(*) " + strings.TrimSpace(fromClause)
	return countQuery
}

// ExecutePaginatedQuery executes a paginated query and returns the results with metadata
// This is a generic helper that can be used by repositories
func ExecutePaginatedQuery[T any](
	ctx context.Context,
	db *sqlx.DB,
	selectQuery string,
	pagination params.PaginationParams,
	args ...any,
) (*PaginationResult[T], error) {
	// Build count query
	countQuery := BuildCountQuery(selectQuery)

	// Execute count query
	var totalCount int
	err := db.GetContext(ctx, &totalCount, countQuery, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			totalCount = 0
		} else {
			return nil, fmt.Errorf("failed to get total count: %w", err)
		}
	}

	// Build paginated query
	paginationClause := BuildPaginationClause(pagination.Offset, pagination.Limit)

	// Add pagination parameters to args
	paginatedArgs := append(args, sql.Named("Offset", pagination.Offset), sql.Named("Limit", pagination.Limit))

	// Execute select query with pagination
	var data []T
	err = db.SelectContext(ctx, &data, selectQuery+" "+paginationClause, paginatedArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute paginated query: %w", err)
	}

	// Calculate total pages
	totalPages := params.CalculateTotalPages(totalCount, pagination.Limit)

	return &PaginationResult[T]{
		Data:       data,
		TotalCount: totalCount,
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		TotalPages: totalPages,
	}, nil
}
