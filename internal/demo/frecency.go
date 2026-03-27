// setup:feature:demo

package demo

import (
	"context"
	"fmt"
	"time"
)

// PageVisit represents a frecency-scored page visit record.
type PageVisit struct {
	Path      string
	Title     string
	Visits    int
	LastVisit time.Time
	Score     float64
}

func (d *DB) initFrecency() error {
	_, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS page_visits (
		session_id TEXT NOT NULL,
		path       TEXT NOT NULL,
		title      TEXT NOT NULL DEFAULT '',
		visits     INTEGER NOT NULL DEFAULT 1,
		last_visit TEXT NOT NULL DEFAULT (datetime('now')),
		PRIMARY KEY (session_id, path)
	)`)
	if err != nil {
		return fmt.Errorf("create page_visits table: %w", err)
	}
	return nil
}

// RecordVisit upserts a page visit for the given session.
func (d *DB) RecordVisit(ctx context.Context, sessionID, path, title string) error {
	_, err := d.db.ExecContext(ctx,
		`INSERT INTO page_visits (session_id, path, title, visits, last_visit)
		 VALUES (?, ?, ?, 1, datetime('now'))
		 ON CONFLICT(session_id, path) DO UPDATE SET
		   visits = visits + 1,
		   last_visit = datetime('now'),
		   title = excluded.title`,
		sessionID, path, title)
	if err != nil {
		return fmt.Errorf("record visit: %w", err)
	}
	return nil
}

// TopFrecent returns the top frecent pages for a session, scored by visits/recency.
func (d *DB) TopFrecent(ctx context.Context, sessionID string, limit int) ([]PageVisit, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT path, title, visits, last_visit,
		        CAST(visits AS REAL) / (julianday('now') - julianday(last_visit) + 1) AS score
		 FROM page_visits
		 WHERE session_id = ?
		 ORDER BY score DESC
		 LIMIT ?`,
		sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("top frecent: %w", err)
	}
	defer rows.Close()

	var results []PageVisit
	for rows.Next() {
		var pv PageVisit
		var lastVisit string
		if err := rows.Scan(&pv.Path, &pv.Title, &pv.Visits, &lastVisit, &pv.Score); err != nil {
			return nil, fmt.Errorf("scan frecent row: %w", err)
		}
		pv.LastVisit, _ = time.Parse("2006-01-02 15:04:05", lastVisit)
		results = append(results, pv)
	}
	return results, rows.Err()
}

// PopularPages returns the most popular pages across all sessions.
func (d *DB) PopularPages(ctx context.Context, limit int) ([]PageVisit, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT path, title, SUM(visits) AS total_visits, MAX(last_visit) AS last_visit,
		        CAST(SUM(visits) AS REAL) / (julianday('now') - julianday(MAX(last_visit)) + 1) AS score
		 FROM page_visits
		 GROUP BY path
		 ORDER BY score DESC
		 LIMIT ?`,
		limit)
	if err != nil {
		return nil, fmt.Errorf("popular pages: %w", err)
	}
	defer rows.Close()

	var results []PageVisit
	for rows.Next() {
		var pv PageVisit
		var lastVisit string
		if err := rows.Scan(&pv.Path, &pv.Title, &pv.Visits, &lastVisit, &pv.Score); err != nil {
			return nil, fmt.Errorf("scan popular row: %w", err)
		}
		pv.LastVisit, _ = time.Parse("2006-01-02 15:04:05", lastVisit)
		results = append(results, pv)
	}
	return results, rows.Err()
}
