// setup:feature:session_settings

package repository

import (
	"context"
	"database/sql"
	"fmt"

	dbrepoManager "catgoose/dothog/internal/database/repository"
	"catgoose/dothog/internal/database/schema"

	"github.com/catgoose/fraggle/dbrepo"
	"github.com/catgoose/porter"
)

// sessionSettingsRepository provides session settings data access.
type sessionSettingsRepository struct {
	repo *dbrepoManager.RepoManager
}

// NewSessionSettingsRepository creates a new session settings repository.
// The returned value satisfies both porter.SessionSettingsProvider and
// routes.SessionSettingsStore via Go's implicit interface satisfaction.
func NewSessionSettingsRepository(repo *dbrepoManager.RepoManager) *sessionSettingsRepository {
	return &sessionSettingsRepository{repo: repo}
}

// selectCols lists the columns matching the porter.SessionSettings struct.
// SessionSettingsTable.SelectColumns() includes CreatedAt which the domain
// struct omits, so we list them explicitly.
var selectCols = dbrepo.Columns("Id", "SessionUUID", "Theme", "Layout", "UpdatedAt")

var tableName = schema.SessionSettingsTable.Name

// GetByUUID returns settings for the given session UUID, or nil if not found.
func (r *sessionSettingsRepository) GetByUUID(ctx context.Context, uuid string) (*porter.SessionSettings, error) {
	w := dbrepo.NewWhere().And("SessionUUID = @SessionUUID", sql.Named("SessionUUID", uuid))
	query, args := dbrepo.NewSelect(tableName, selectCols).Where(w).Build()

	var s porter.SessionSettings
	err := r.repo.GetDB().GetContext(ctx, &s, query, args...)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session settings: %w", err)
	}
	return &s, nil
}

// Upsert inserts or updates session settings by SessionUUID.
func (r *sessionSettingsRepository) Upsert(ctx context.Context, s *porter.SessionSettings) error {
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
		_, err = r.repo.GetDB().ExecContext(ctx, query,
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

	insertCols := schema.SessionSettingsTable.InsertColumns()
	query := dbrepo.InsertInto(tableName, insertCols...)
	var createdAt = dbrepo.GetNow()
	dbrepo.SetUpdateTimestamp(&s.UpdatedAt)
	_, err = r.repo.GetDB().ExecContext(ctx, query,
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

// Touch updates UpdatedAt for the given session UUID.
func (r *sessionSettingsRepository) Touch(ctx context.Context, uuid string) error {
	query := fmt.Sprintf("UPDATE %s SET %s WHERE SessionUUID = @SessionUUID",
		tableName,
		dbrepo.SetClause("UpdatedAt"),
	)
	now := dbrepo.GetNow()
	_, err := r.repo.GetDB().ExecContext(ctx, query,
		sql.Named("UpdatedAt", now),
		sql.Named("SessionUUID", uuid),
	)
	if err != nil {
		return fmt.Errorf("touch session settings: %w", err)
	}
	return nil
}

// ListAll returns all session settings rows ordered by most recently updated.
func (r *sessionSettingsRepository) ListAll(ctx context.Context) ([]porter.SessionSettings, error) {
	query, args := dbrepo.NewSelect(tableName, selectCols).OrderBy("UpdatedAt DESC").Build()
	var rows []porter.SessionSettings
	err := r.repo.GetDB().SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list session settings: %w", err)
	}
	return rows, nil
}

