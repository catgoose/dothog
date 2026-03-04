// setup:feature:demo

// Package demo provides a self-contained SQLite database for the /demo route.
package demo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver with database/sql
)

// Item represents one inventory row returned from the demo database.
type Item struct {
	Name      string
	Category  string
	CreatedAt string
	ID        int
	Price     float64
	Stock     int
	Active    bool
}

// DB wraps a sqlite3 connection.
type DB struct{ db *sql.DB }

// allowedSort maps query-param sort keys to safe SQL column names.
var allowedSort = map[string]string{
	"name":     "name",
	"category": "category",
	"price":    "price",
	"stock":    "stock",
}

// Open opens (or creates) a SQLite file at path, initialises the schema, and
// seeds it if empty.
func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	d := &DB{db: db}
	if err := d.initSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return d, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) initSchema() error {
	_, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS items (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT    NOT NULL,
		category   TEXT    NOT NULL,
		price      REAL    NOT NULL,
		stock      INTEGER NOT NULL,
		active     INTEGER NOT NULL DEFAULT 1,
		created_at TEXT    NOT NULL
	)`)
	if err != nil {
		return err
	}
	var count int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return d.seed()
	}
	return nil
}

type seedRow struct {
	name, category, createdAt string
	price                     float64
	stock                     int
	active                    bool
}

func (d *DB) seed() error {
	rows := []seedRow{
		{"Laptop Pro 15", "Electronics", "2024-01-05", 1299.99, 12, true},
		{"Wireless Earbuds", "Electronics", "2024-01-08", 89.95, 45, true},
		{"Smart Watch", "Electronics", "2024-01-12", 249.00, 8, true},
		{"USB-C Hub", "Electronics", "2024-01-15", 49.99, 30, true},
		{"Mechanical Keyboard", "Electronics", "2024-01-20", 129.00, 0, false},
		{"4K Monitor", "Electronics", "2024-01-22", 399.99, 5, true},
		{"Webcam HD", "Electronics", "2024-02-01", 79.95, 20, false},
		{"SSD 1TB", "Electronics", "2024-02-05", 99.00, 15, true},
		{"Noise Cancelling Headphones", "Electronics", "2024-02-10", 199.00, 6, true},
		{"Portable Charger", "Electronics", "2024-02-14", 39.99, 50, true},
		{"Running Shoes", "Sports", "2024-01-06", 119.99, 22, true},
		{"Yoga Mat", "Sports", "2024-01-09", 34.95, 35, true},
		{"Dumbbell Set", "Sports", "2024-01-13", 89.99, 10, true},
		{"Cycling Gloves", "Sports", "2024-01-16", 24.95, 40, false},
		{"Water Bottle", "Sports", "2024-01-21", 19.99, 60, true},
		{"Jump Rope", "Sports", "2024-01-25", 14.99, 25, true},
		{"Resistance Bands", "Sports", "2024-02-02", 29.95, 18, true},
		{"Foam Roller", "Sports", "2024-02-06", 27.99, 12, false},
		{"Sports Socks 3-Pack", "Sports", "2024-02-11", 12.99, 80, true},
		{"Gym Bag", "Sports", "2024-02-15", 44.95, 7, true},
		{"Go Programming Language", "Books", "2024-01-07", 39.99, 14, true},
		{"Clean Code", "Books", "2024-01-10", 34.95, 9, true},
		{"HTMX In Practice", "Books", "2024-01-14", 29.99, 20, true},
		{"Database Design", "Books", "2024-01-17", 44.99, 5, false},
		{"Algorithms Unlocked", "Books", "2024-01-23", 32.00, 11, true},
		{"Design Patterns", "Books", "2024-01-26", 49.95, 3, true},
		{"The Pragmatic Programmer", "Books", "2024-02-03", 39.99, 16, true},
		{"Structure and Interpretation", "Books", "2024-02-07", 54.95, 0, false},
		{"Refactoring", "Books", "2024-02-12", 37.00, 8, true},
		{"Modern CSS", "Books", "2024-02-16", 27.99, 21, true},
		{"Cotton T-Shirt", "Clothing", "2024-01-03", 19.99, 50, true},
		{"Denim Jacket", "Clothing", "2024-01-11", 74.95, 14, true},
		{"Merino Wool Socks", "Clothing", "2024-01-18", 16.99, 30, true},
		{"Canvas Backpack", "Clothing", "2024-01-24", 59.95, 7, false},
		{"Baseball Cap", "Clothing", "2024-01-27", 22.00, 25, true},
		{"Rain Jacket", "Clothing", "2024-02-04", 89.99, 9, true},
		{"Leather Belt", "Clothing", "2024-02-08", 29.95, 18, true},
		{"Winter Gloves", "Clothing", "2024-02-13", 24.99, 0, false},
		{"Flannel Shirt", "Clothing", "2024-02-17", 44.95, 12, true},
		{"Linen Trousers", "Clothing", "2024-02-20", 54.99, 6, true},
		{"Organic Coffee Beans 1kg", "Food", "2024-01-04", 22.95, 40, true},
		{"Dark Chocolate Bar", "Food", "2024-01-19", 4.99, 100, true},
		{"Olive Oil 500ml", "Food", "2024-01-28", 12.50, 35, true},
		{"Granola Mix", "Food", "2024-02-09", 8.99, 55, false},
		{"Green Tea 50 bags", "Food", "2024-01-15", 9.95, 60, true},
		{"Hot Sauce Collection", "Food", "2024-02-18", 19.99, 20, true},
		{"Protein Powder 1kg", "Food", "2024-01-30", 34.95, 15, true},
		{"Almond Butter 500g", "Food", "2024-02-19", 13.50, 28, true},
		{"Rice Crackers Pack", "Food", "2024-01-08", 3.99, 80, false},
		{"Kombucha 6-Pack", "Food", "2024-02-21", 14.95, 24, true},
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(
		`INSERT INTO items (name, category, price, stock, active, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, row := range rows {
		active := 0
		if row.active {
			active = 1
		}
		if _, err := stmt.Exec(row.name, row.category, row.price, row.stock, active, row.createdAt); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// CreateItem inserts a new item and returns it with the assigned ID.
func (d *DB) CreateItem(ctx context.Context, item Item) (Item, error) {
	res, err := d.db.ExecContext(ctx,
		`INSERT INTO items (name, category, price, stock, active, created_at) VALUES (?, ?, ?, ?, ?, date('now'))`,
		item.Name, item.Category, item.Price, item.Stock, boolToInt(item.Active))
	if err != nil {
		return Item{}, fmt.Errorf("create item: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Item{}, fmt.Errorf("get last insert id: %w", err)
	}
	item.ID = int(id)
	return item, nil
}

// GetItem returns a single Item by ID.
func (d *DB) GetItem(ctx context.Context, id int) (Item, error) {
	var item Item
	var activeInt int
	err := d.db.QueryRowContext(ctx,
		"SELECT id, name, category, price, stock, active, created_at FROM items WHERE id = ?", id).
		Scan(&item.ID, &item.Name, &item.Category, &item.Price, &item.Stock, &activeInt, &item.CreatedAt)
	if err != nil {
		return Item{}, fmt.Errorf("get item %d: %w", id, err)
	}
	item.Active = activeInt != 0
	return item, nil
}

// UpdateItem updates name, category, price, stock, and active for the given item.
func (d *DB) UpdateItem(ctx context.Context, item Item) error {
	_, err := d.db.ExecContext(ctx,
		"UPDATE items SET name=?, category=?, price=?, stock=?, active=? WHERE id=?",
		item.Name, item.Category, item.Price, item.Stock, boolToInt(item.Active), item.ID)
	if err != nil {
		return fmt.Errorf("update item %d: %w", item.ID, err)
	}
	return nil
}

// DeleteItem deletes an item by ID.
func (d *DB) DeleteItem(ctx context.Context, id int) error {
	_, err := d.db.ExecContext(ctx, "DELETE FROM items WHERE id=?", id)
	if err != nil {
		return fmt.Errorf("delete item %d: %w", id, err)
	}
	return nil
}

// ListItems queries items with optional filters, sort, and pagination.
// Returns the matching items, total row count (ignoring pagination), and any error.
func (d *DB) ListItems(ctx context.Context, q, category, active, sortBy, sortDir string, page, perPage int) ([]Item, int, error) {
	col, ok := allowedSort[sortBy]
	if !ok {
		col = "name"
		sortDir = "asc"
	}
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "asc"
	}

	var conditions []string
	var args []any

	if q != "" {
		conditions = append(conditions, "name LIKE ?")
		args = append(args, "%"+q+"%")
	}
	if category != "" {
		conditions = append(conditions, "category = ?")
		args = append(args, category)
	}
	if active == "true" {
		conditions = append(conditions, "active = 1")
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM items %s", where)
	if err := d.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count query: %w", err)
	}

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * perPage

	// col and sortDir are validated against the allowedSort map and "asc"/"desc" check above.
	query := fmt.Sprintf(
		"SELECT id, name, category, price, stock, active, created_at FROM items %s ORDER BY %s %s LIMIT ? OFFSET ?",
		where, col, sortDir,
	)
	listArgs := make([]any, len(args), len(args)+2)
	copy(listArgs, args)
	listArgs = append(listArgs, perPage, offset)

	dbRows, err := d.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list query: %w", err)
	}
	defer dbRows.Close()

	var items []Item
	for dbRows.Next() {
		var item Item
		var activeInt int
		if err := dbRows.Scan(&item.ID, &item.Name, &item.Category, &item.Price, &item.Stock, &activeInt, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		item.Active = activeInt != 0
		items = append(items, item)
	}
	return items, total, dbRows.Err()
}
