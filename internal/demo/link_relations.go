// setup:feature:demo
package demo

import "fmt"

// StoredLinkRelation represents an admin-created link relation persisted in SQLite.
type StoredLinkRelation struct {
	ID        int
	Source    string
	Rel       string
	Target    string
	Title     string
	GroupName string
	CreatedAt string
}

func (d *DB) initLinkRelations() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS link_relations (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			source     TEXT NOT NULL,
			rel        TEXT NOT NULL DEFAULT 'related',
			target     TEXT NOT NULL,
			title      TEXT NOT NULL,
			group_name TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(source, rel, target)
		)
	`)
	return err
}

// ListStoredLinks returns all admin-created link relations.
func (d *DB) ListStoredLinks() ([]StoredLinkRelation, error) {
	rows, err := d.db.Query(
		"SELECT id, source, rel, target, title, group_name, created_at FROM link_relations ORDER BY source, rel, target")
	if err != nil {
		return nil, fmt.Errorf("list stored links: %w", err)
	}
	defer rows.Close()

	var links []StoredLinkRelation
	for rows.Next() {
		var l StoredLinkRelation
		if err := rows.Scan(&l.ID, &l.Source, &l.Rel, &l.Target, &l.Title, &l.GroupName, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan stored link: %w", err)
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

// InsertLink creates a new admin link relation.
func (d *DB) InsertLink(source, rel, target, title, groupName string) error {
	_, err := d.db.Exec(
		"INSERT INTO link_relations (source, rel, target, title, group_name) VALUES (?, ?, ?, ?, ?)",
		source, rel, target, title, groupName)
	if err != nil {
		return fmt.Errorf("insert link: %w", err)
	}
	return nil
}

// DeleteLink removes an admin-created link relation by ID.
func (d *DB) DeleteLink(id int) error {
	res, err := d.db.Exec("DELETE FROM link_relations WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete link %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("delete link %d: no rows affected", id)
	}
	return nil
}

// StoredLinkIDs returns the set of (source, rel, target) tuples that are admin-created,
// keyed for quick lookup.
func (d *DB) StoredLinkIDs() (map[int]bool, error) {
	rows, err := d.db.Query("SELECT id FROM link_relations")
	if err != nil {
		return nil, fmt.Errorf("stored link ids: %w", err)
	}
	defer rows.Close()

	ids := make(map[int]bool)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}
