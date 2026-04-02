// setup:feature:sync
package routes

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
)

// knownTables is the set of tables that support version-based sync.
// This is the single source of truth for SQL table name validation —
// no regex needed when we can enumerate the valid values.
var knownTables = map[string]bool{
	"Tasks":  true,
	"Items":  true,
	"People": true,
}

// VersionChecker looks up the current version of a row by table and ID.
type VersionChecker interface {
	CurrentVersion(ctx context.Context, table string, id int) (int, error)
}

// SQLVersionChecker checks row versions against a SQL database.
// It expects tables to have an ID column and a Version column (as created by
// chuck's WithVersion() schema trait).
type SQLVersionChecker struct {
	db *sql.DB
}

// NewSQLVersionChecker creates a version checker backed by the given database.
func NewSQLVersionChecker(db *sql.DB) *SQLVersionChecker {
	return &SQLVersionChecker{db: db}
}

// CurrentVersion returns the current version of a row, or -1 if not found.
func (vc *SQLVersionChecker) CurrentVersion(ctx context.Context, table string, id int) (int, error) {
	if !knownTables[table] {
		return 0, fmt.Errorf("unknown table: %s", table)
	}

	var version int
	query := fmt.Sprintf("SELECT Version FROM %s WHERE ID = ? AND DeletedAt IS NULL", table)
	err := vc.db.QueryRowContext(ctx, query, id).Scan(&version)
	if err == sql.ErrNoRows {
		return -1, nil // Row not found (deleted or never existed)
	}
	if err != nil {
		return 0, fmt.Errorf("check version for %s/%d: %w", table, id, err)
	}
	return version, nil
}

// parseResourceURL extracts a table name hint and row ID from a sync operation URL.
// Expected format: /demo/repository/{resource}/{id} or /demo/{resource}/{id}
// Returns empty table and 0 id if the URL doesn't match a known pattern.
var resourceURLPattern = regexp.MustCompile(`/demo/(?:repository/)?(\w+)/(\d+)$`)

func parseResourceURL(url string) (table string, id int, ok bool) {
	// Match patterns like /demo/repository/tasks/42 or /demo/items/7
	matches := resourceURLPattern.FindStringSubmatch(url)
	if len(matches) != 3 {
		return "", 0, false
	}

	resource := matches[1]
	rowID, err := strconv.Atoi(matches[2])
	if err != nil {
		return "", 0, false
	}

	// Map URL resource names to table names
	tableMap := map[string]string{
		"tasks":  "Tasks",
		"items":  "Items",
		"people": "People",
	}

	tableName, exists := tableMap[resource]
	if !exists {
		return "", 0, false
	}

	return tableName, rowID, true
}
