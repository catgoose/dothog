// Package params provides utilities for parsing HTTP request parameters including pagination and filters.
package params

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

// PaginationParams holds parsed pagination parameters
type PaginationParams struct {
	Page   int
	Limit  int
	Offset int
}

// ParsePaginationParams parses pagination parameters from query params with defaults
func ParsePaginationParams(c echo.Context, defaultLimit, maxLimit int) PaginationParams {
	pageStr := c.QueryParam("page")
	limitStr := c.QueryParam("limit")

	page := 1
	limit := defaultLimit

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= maxLimit {
			limit = l
		}
	}

	offset := (page - 1) * limit

	return PaginationParams{
		Page:   page,
		Limit:  limit,
		Offset: offset,
	}
}

// CalculateTotalPages calculates total pages from total count and limit
func CalculateTotalPages(totalCount, limit int) int {
	if limit <= 0 {
		return 0
	}
	return (totalCount + limit - 1) / limit
}

// ParseRequestID parses and validates a request ID from path parameter or query parameter
func ParseRequestID(c echo.Context) (int, error) {
	id := c.Param("id")
	if id == "" {
		id = c.QueryParam("id")
	}
	if id == "" {
		return 0, fmt.Errorf("id parameter not found")
	}
	parsedID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("invalid id: %w", err)
	}
	return parsedID, nil
}

// FilterParams holds parsed filter and pagination parameters
type FilterParams struct {
	Search       string
	Status       string
	DurationType string
	Sort         string
	Pagination   PaginationParams
	Year         int
}

// ParseFilterParams extracts and parses all filter query parameters from echo.Context
func ParseFilterParams(c echo.Context, defaultLimit, maxLimit int) FilterParams {
	search := c.QueryParam("search")
	yearStr := c.QueryParam("year")
	status := c.QueryParam("status")
	durationType := c.QueryParam("type")
	sort := c.QueryParam("sort")

	pagination := ParsePaginationParams(c, defaultLimit, maxLimit)

	year := 0
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y > 0 {
			year = y
		}
	}

	return FilterParams{
		Search:       search,
		Year:         year,
		Status:       status,
		DurationType: durationType,
		Pagination:   pagination,
		Sort:         sort,
	}
}

// ResolveYearWithDefault handles year parsing and default year selection logic
func ResolveYearWithDefault(c echo.Context, year int, availableYears []int) int {
	yearParamExists := c.Request().URL.Query().Has("year")
	if year == 0 && !yearParamExists && len(availableYears) > 0 {
		return availableYears[0]
	}
	return year
}

// SortColumn represents a single sort column with direction
type SortColumn struct {
	Column    string
	Direction string
}

// ParseSortString parses a sort string in the format "column1:direction,column2:direction" into SortColumn slices
func ParseSortString(sortStr string) []SortColumn {
	if sortStr == "" {
		return nil
	}

	var sorts []SortColumn
	for part := range strings.SplitSeq(sortStr, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		colonIdx := strings.LastIndex(part, ":")
		if colonIdx == -1 {
			continue
		}
		column := strings.ToLower(strings.TrimSpace(part[:colonIdx]))
		direction := strings.ToLower(strings.TrimSpace(part[colonIdx+1:]))
		if column != "" && (direction == "asc" || direction == "desc") {
			sorts = append(sorts, SortColumn{Column: column, Direction: direction})
		}
	}
	return sorts
}

// BuildSortString builds a sort string from SortColumn slices
func BuildSortString(sorts []SortColumn) string {
	if len(sorts) == 0 {
		return ""
	}
	var parts []string
	for _, sort := range sorts {
		parts = append(parts, fmt.Sprintf("%s:%s", sort.Column, sort.Direction))
	}
	return strings.Join(parts, ",")
}
