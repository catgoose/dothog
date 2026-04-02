// setup:feature:demo

package demo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// SeedDB provides access to the seed.db reference data via ATTACH.
type SeedDB struct {
	db       *sql.DB
	attached bool
}

// SeedTableInfo describes a table in the seed database.
type SeedTableInfo struct {
	Name     string
	RowCount int
}

// NewSeedDB wraps an existing in-memory database connection for seed operations.
func NewSeedDB(db *sql.DB) *SeedDB {
	return &SeedDB{db: db}
}

// Attach attaches seed.db as the "seed" schema. Safe to call multiple times.
func (s *SeedDB) Attach(ctx context.Context, seedPath string) error {
	if s.attached {
		return nil
	}
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("ATTACH DATABASE '%s' AS seed", seedPath))
	if err != nil {
		return fmt.Errorf("attach seed.db: %w", err)
	}
	s.attached = true
	return nil
}

// Detach detaches the seed schema.
func (s *SeedDB) Detach(ctx context.Context) error {
	if !s.attached {
		return nil
	}
	_, err := s.db.ExecContext(ctx, "DETACH DATABASE seed")
	if err != nil {
		return fmt.Errorf("detach seed.db: %w", err)
	}
	s.attached = false
	return nil
}

// IsAttached reports whether seed.db is currently attached.
func (s *SeedDB) IsAttached() bool {
	return s.attached
}

// SeedTables returns information about all tables in the seed database.
func (s *SeedDB) SeedTables(ctx context.Context) ([]SeedTableInfo, error) {
	if !s.attached {
		return nil, fmt.Errorf("seed.db not attached")
	}
	names, err := s.listTableNames(ctx, "seed")
	if err != nil {
		return nil, err
	}
	return s.countTables(ctx, "seed", names)
}

// CopyTable copies a table from the seed schema into the main database.
// It creates the table in the main schema if it doesn't exist, then inserts all rows.
func (s *SeedDB) CopyTable(ctx context.Context, tableName string) (int64, error) {
	if !s.attached {
		return 0, fmt.Errorf("seed.db not attached")
	}

	// Get column info from seed table
	cols, err := s.tableColumns(ctx, "seed", tableName)
	if err != nil {
		return 0, err
	}

	// Get the CREATE TABLE statement from the seed
	var createSQL string
	err = s.db.QueryRowContext(ctx,
		"SELECT sql FROM seed.sqlite_master WHERE type='table' AND name=?", tableName,
	).Scan(&createSQL)
	if err != nil {
		return 0, fmt.Errorf("get create sql for %s: %w", tableName, err)
	}

	// Create in main schema (IF NOT EXISTS)
	createSQL = strings.Replace(createSQL, "CREATE TABLE", "CREATE TABLE IF NOT EXISTS", 1)
	if _, err := s.db.ExecContext(ctx, createSQL); err != nil {
		return 0, fmt.Errorf("create table %s: %w", tableName, err)
	}

	// Insert from seed
	colList := strings.Join(cols, ", ")
	insertSQL := fmt.Sprintf("INSERT OR IGNORE INTO main.%s (%s) SELECT %s FROM seed.%s",
		tableName, colList, colList, tableName)

	result, err := s.db.ExecContext(ctx, insertSQL)
	if err != nil {
		return 0, fmt.Errorf("copy data for %s: %w", tableName, err)
	}

	return result.RowsAffected()
}

// tableColumns returns the column names for a table in the given schema.
func (s *SeedDB) tableColumns(ctx context.Context, schema, table string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA %s.table_info(%s)", schema, table))
	if err != nil {
		return nil, fmt.Errorf("table_info %s.%s: %w", schema, table, err)
	}
	defer func() { _ = rows.Close() }()

	var cols []string
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

// MainTables returns information about all user tables in the main database.
func (s *SeedDB) MainTables(ctx context.Context) ([]SeedTableInfo, error) {
	names, err := s.listTableNames(ctx, "main")
	if err != nil {
		return nil, err
	}
	return s.countTables(ctx, "main", names)
}

// listTableNames returns table names for the given schema, closing the cursor
// before returning so a single-connection pool isn't blocked.
func (s *SeedDB) listTableNames(ctx context.Context, schema string) ([]string, error) {
	query := fmt.Sprintf(
		"SELECT name FROM %s.sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%%' ORDER BY name",
		schema)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list %s tables: %w", schema, err)
	}
	defer func() { _ = rows.Close() }()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// countTables queries row counts for each table, one at a time.
func (s *SeedDB) countTables(ctx context.Context, schema string, names []string) ([]SeedTableInfo, error) {
	tables := make([]SeedTableInfo, 0, len(names))
	for _, name := range names {
		var count int
		if err := s.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", schema, name)).Scan(&count); err != nil {
			return nil, fmt.Errorf("count %s.%s: %w", schema, name, err)
		}
		tables = append(tables, SeedTableInfo{Name: name, RowCount: count})
	}
	return tables, nil
}

// DropMainTable drops a table from the main database.
func (s *SeedDB) DropMainTable(ctx context.Context, tableName string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS main.%s", tableName))
	if err != nil {
		return fmt.Errorf("drop table %s: %w", tableName, err)
	}
	return nil
}

// ExecSQL executes arbitrary SQL against the main database. Returns rows affected.
func (s *SeedDB) ExecSQL(ctx context.Context, query string) (int64, error) {
	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
