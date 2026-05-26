// setup:feature:session_settings

package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/catgoose/chuck/dbrepo"
	"github.com/jmoiron/sqlx"

	"catgoose/dothog/internal/session"
)

// SessionSettingsRepository is framework-owned session-store infrastructure
// (NOT a derived-app starter example — see internal/dbschema for the
// app-data repository pattern). It persists session.Settings rows and
// satisfies both session.SettingsProvider (middleware hydration) and
// session.SettingsAdmin (admin/management routes) via implicit interface
// satisfaction.
type SessionSettingsRepository struct {
	db *sqlx.DB
}

// NewSessionSettingsRepository binds db; the result satisfies both
// session.SettingsProvider and session.SettingsAdmin via implicit interface
// satisfaction, so it can be wired into session.Middleware and into the
// route-side Deps struct without further plumbing.
func NewSessionSettingsRepository(db *sqlx.DB) *SessionSettingsRepository {
	return &SessionSettingsRepository{db: db}
}

// selectCols lists the columns matching the session.Settings struct.
// session.SettingsTable.SelectColumns() includes CreatedAt which the domain
// struct omits, so we list them explicitly.
var selectCols = dbrepo.Columns("Id", "SessionUUID", "Theme", "Layout", "UpdatedAt")

var tableName = session.SettingsTable.Name

// GetByUUID maps sql.ErrNoRows to (nil, nil); other errors are wrapped.
func (r *SessionSettingsRepository) GetByUUID(ctx context.Context, uuid string) (*session.Settings, error) {
	w := dbrepo.NewWhere().And("SessionUUID = @SessionUUID", sql.Named("SessionUUID", uuid))
	query, args := dbrepo.NewSelect(tableName, selectCols).Where(w).Build()

	var s session.Settings
	err := r.db.GetContext(ctx, &s, query, args...)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session settings: %w", err)
	}
	return &s, nil
}

// Upsert keys on SessionUUID; UpdatedAt is bumped on every call.
func (r *SessionSettingsRepository) Upsert(ctx context.Context, s *session.Settings) error {
	existing, err := r.GetByUUID(ctx, s.SessionUUID)
	if err != nil {
		return fmt.Errorf("lookup existing session settings: %w", err)
	}
	if existing != nil {
		query := fmt.Sprintf("UPDATE %s SET %s WHERE SessionUUID = @SessionUUID",
			tableName,
			dbrepo.SetClause("Theme", "Layout", "UpdatedAt"),
		)
		dbrepo.SetUpdateTimestamp(&s.UpdatedAt)
		_, err = r.db.ExecContext(ctx, query,
			sql.Named("Theme", s.Theme),
			sql.Named("Layout", s.Layout),
			sql.Named("UpdatedAt", s.UpdatedAt),
			sql.Named("SessionUUID", s.SessionUUID),
		)
		if err != nil {
			return fmt.Errorf("update session settings: %w", err)
		}
		return nil
	}

	insertCols := session.SettingsTable.InsertColumns()
	query := dbrepo.InsertInto(tableName, insertCols...)
	var createdAt = dbrepo.GetNow()
	dbrepo.SetUpdateTimestamp(&s.UpdatedAt)
	_, err = r.db.ExecContext(ctx, query,
		sql.Named("SessionUUID", s.SessionUUID),
		sql.Named("Theme", s.Theme),
		sql.Named("Layout", s.Layout),
		sql.Named("CreatedAt", createdAt),
		sql.Named("UpdatedAt", s.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert session settings: %w", err)
	}
	return nil
}

// Touch bumps UpdatedAt only; other fields are untouched. No-op if uuid is unknown.
func (r *SessionSettingsRepository) Touch(ctx context.Context, uuid string) error {
	query := fmt.Sprintf("UPDATE %s SET %s WHERE SessionUUID = @SessionUUID",
		tableName,
		dbrepo.SetClause("UpdatedAt"),
	)
	now := dbrepo.GetNow()
	_, err := r.db.ExecContext(ctx, query,
		sql.Named("UpdatedAt", now),
		sql.Named("SessionUUID", uuid),
	)
	if err != nil {
		return fmt.Errorf("touch session settings: %w", err)
	}
	return nil
}

// ListAll enumerates every row, ordered by UpdatedAt DESC.
func (r *SessionSettingsRepository) ListAll(ctx context.Context) ([]session.Settings, error) {
	query, args := dbrepo.NewSelect(tableName, selectCols).OrderBy("UpdatedAt DESC").Build()
	var rows []session.Settings
	err := r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list session settings: %w", err)
	}
	return rows, nil
}

// UpdateThemeByUUID rewrites Theme only. UpdatedAt stays request-activity
// owned by session.Middleware, so admin theme edits do not make a session
// look newly active. The bool reports whether a row matched.
func (r *SessionSettingsRepository) UpdateThemeByUUID(ctx context.Context, uuid, theme string) (bool, error) {
	query := fmt.Sprintf("UPDATE %s SET %s WHERE SessionUUID = @SessionUUID",
		tableName,
		dbrepo.SetClause("Theme"),
	)
	res, err := r.db.ExecContext(ctx, query,
		sql.Named("Theme", theme),
		sql.Named("SessionUUID", uuid),
	)
	if err != nil {
		return false, fmt.Errorf("update session theme: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected session theme: %w", err)
	}
	return n > 0, nil
}

// DeleteByUUID removes the row matching uuid; missing rows are not an error
// so admin handlers can stay idempotent across duplicate clicks and stale
// fragment refreshes.
func (r *SessionSettingsRepository) DeleteByUUID(ctx context.Context, uuid string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE SessionUUID = @SessionUUID", tableName)
	_, err := r.db.ExecContext(ctx, query,
		sql.Named("SessionUUID", uuid),
	)
	if err != nil {
		return fmt.Errorf("delete session settings: %w", err)
	}
	return nil
}
