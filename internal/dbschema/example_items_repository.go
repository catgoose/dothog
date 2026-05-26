// setup:feature:database

package dbschema

// EXAMPLE STARTER FILE — copy and adapt.
//
// This file is the dothog starter example for the chuck-backed app-data
// repository pattern. It exists so derived apps that select MSSQL or
// PostgreSQL inherit a concrete reference to copy from. The Items table is
// intentionally generic; rename / replace it with your own table and
// remove this file once you've authored your real repositories.
//
// What the example shows:
//
//  1. A schema.TableDef declared alongside the repository (ItemsTable).
//  2. A concrete repository struct that holds *sqlx.DB directly — no
//     "manager as runtime access point" indirection.
//  3. Single-statement queries via GetContext / ExecContext / SelectContext.
//  4. Multi-statement work via database.WithTransaction(ctx, db, fn).
//  5. chuck dbrepo helpers (NewSelect, NewWhere, SetClause, InsertInto,
//     Columns) for query construction.
//
// What to do next:
//
//   - Register ItemsTable in Tables() above (or replace it with your own).
//   - Wire NewItemRepository(dbx) into main.go alongside any other
//     app-data repositories you build.
//   - Delete this file when your real app-data code is in place.

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"catgoose/dothog/internal/database"
	"catgoose/dothog/internal/database/schema"

	"github.com/catgoose/chuck/dbrepo"
	"github.com/jmoiron/sqlx"
)

// ItemsTable is the example app-data table. Replace with your own; register
// it in Tables() above so schema.Materializer ensures it on startup.
var ItemsTable = schema.NewTable("Items").
	Columns(
		schema.AutoIncrCol("Id"),
		schema.Col("Name", schema.TypeString(255)).NotNull(),
		schema.Col("Quantity", schema.TypeInt()).NotNull().Default("0"),
	).
	WithTimestamps()

// Item is the example domain shape mirroring ItemsTable.
type Item struct {
	UpdatedAt time.Time `db:"UpdatedAt" json:"updatedAt"`
	CreatedAt time.Time `db:"CreatedAt" json:"createdAt"`
	Name      string    `db:"Name" json:"name"`
	ID        int       `db:"Id" json:"id"`
	Quantity  int       `db:"Quantity" json:"quantity"`
}

// ErrItemNotFound is returned when an Item lookup matches zero rows.
var ErrItemNotFound = fmt.Errorf("item not found")

// ItemRepository is the example repository. Hold *sqlx.DB directly; reach
// transactions via database.WithTransaction.
type ItemRepository struct {
	db *sqlx.DB
}

// NewItemRepository binds db.
func NewItemRepository(db *sqlx.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

// GetByID returns the row matching id; sql.ErrNoRows becomes ErrItemNotFound.
func (r *ItemRepository) GetByID(ctx context.Context, id int) (*Item, error) {
	cols := dbrepo.Columns(ItemsTable.SelectColumns()...)
	w := dbrepo.NewWhere().And("Id = @Id", sql.Named("Id", id))
	query, args := dbrepo.NewSelect(ItemsTable.Name, cols).Where(w).Build()

	var item Item
	if err := r.db.GetContext(ctx, &item, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("get item: %w", err)
	}
	return &item, nil
}

// ListAll enumerates every row ordered by Id.
func (r *ItemRepository) ListAll(ctx context.Context) ([]Item, error) {
	cols := dbrepo.Columns(ItemsTable.SelectColumns()...)
	query, args := dbrepo.NewSelect(ItemsTable.Name, cols).OrderBy("Id ASC").Build()

	var items []Item
	if err := r.db.SelectContext(ctx, &items, query, args...); err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	return items, nil
}

// Create inserts a new row. item.ID is NOT populated by this method: each
// supported app-data engine returns generated keys differently (SQLite uses
// last_insert_rowid(), MSSQL SCOPE_IDENTITY(), PostgreSQL RETURNING), so this
// starter intentionally stays plain. If you need the generated id, follow up
// with a dialect-aware query (e.g. SELECT Id FROM Items WHERE … ORDER BY …
// or a RETURNING/OUTPUT clause) in your own repository.
func (r *ItemRepository) Create(ctx context.Context, item *Item) error {
	dbrepo.SetCreateTimestamps(&item.CreatedAt, &item.UpdatedAt)
	insertCols := ItemsTable.InsertColumns()
	query := dbrepo.InsertInto(ItemsTable.Name, insertCols...)

	if _, err := r.db.ExecContext(ctx, query,
		sql.Named("Name", item.Name),
		sql.Named("Quantity", item.Quantity),
		sql.Named("CreatedAt", item.CreatedAt),
		sql.Named("UpdatedAt", item.UpdatedAt),
	); err != nil {
		return fmt.Errorf("create item: %w", err)
	}
	return nil
}

// Update mutates the row matching item.ID; ErrItemNotFound when no row matches.
func (r *ItemRepository) Update(ctx context.Context, item *Item) error {
	dbrepo.SetUpdateTimestamp(&item.UpdatedAt)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE Id = @Id",
		ItemsTable.Name,
		dbrepo.SetClause("Name", "Quantity", "UpdatedAt"),
	)
	result, err := r.db.ExecContext(ctx, query,
		sql.Named("Id", item.ID),
		sql.Named("Name", item.Name),
		sql.Named("Quantity", item.Quantity),
		sql.Named("UpdatedAt", item.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("update item: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrItemNotFound
	}
	return nil
}

// Delete removes the row matching id; missing rows return ErrItemNotFound.
func (r *ItemRepository) Delete(ctx context.Context, id int) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE Id = @Id", ItemsTable.Name)
	result, err := r.db.ExecContext(ctx, query, sql.Named("Id", id))
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrItemNotFound
	}
	return nil
}

// IncrementInTransaction is the canonical multi-statement-work example: it
// reads the current Quantity, adds delta, and writes back — all under one
// transaction via database.WithTransaction.
func (r *ItemRepository) IncrementInTransaction(ctx context.Context, id int, delta int) error {
	return database.WithTransaction(ctx, r.db, func(ctx context.Context, tx *sqlx.Tx) error {
		var current int
		if err := tx.GetContext(ctx, &current, "SELECT Quantity FROM "+ItemsTable.Name+" WHERE Id = @Id", sql.Named("Id", id)); err != nil {
			if err == sql.ErrNoRows {
				return ErrItemNotFound
			}
			return fmt.Errorf("read quantity: %w", err)
		}
		_, err := tx.ExecContext(ctx,
			"UPDATE "+ItemsTable.Name+" SET Quantity = @Q, UpdatedAt = @U WHERE Id = @Id",
			sql.Named("Q", current+delta),
			sql.Named("U", dbrepo.GetNow()),
			sql.Named("Id", id),
		)
		if err != nil {
			return fmt.Errorf("update quantity: %w", err)
		}
		return nil
	})
}
