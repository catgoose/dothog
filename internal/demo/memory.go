// setup:feature:demo

package demo

import (
	"database/sql"
	"fmt"

	_ "github.com/catgoose/chuck/driver/sqlite" // register SQLite driver
)

// OpenMemoryDB opens a new in-memory SQLite database for admin/seed operations.
// It limits to 1 connection so that ATTACH and all queries share the same
// underlying SQLite connection (ATTACH is per-connection state).
func OpenMemoryDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("open in-memory db: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping in-memory db: %w", err)
	}
	return db, nil
}
