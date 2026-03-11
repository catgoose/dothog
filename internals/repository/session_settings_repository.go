// setup:feature:session_settings

package repository

import (
	"context"
	"database/sql"
	"fmt"

	dbrepo "catgoose/harmony/internals/database/repository"
	"catgoose/harmony/internals/database/schema"
	"catgoose/harmony/internals/domain"
)

// SessionSettingsRepository defines operations for session settings data access.
type SessionSettingsRepository interface {
	GetByUUID(ctx context.Context, uuid string) (*domain.SessionSettings, error)
	Upsert(ctx context.Context, s *domain.SessionSettings) error
	Touch(ctx context.Context, uuid string) error
	DeleteStale(ctx context.Context, days int) (int64, error)
}

// sessionSettingsRepository implements SessionSettingsRepository.
type sessionSettingsRepository struct {
	repo *dbrepo.RepoManager
}

// NewSessionSettingsRepository creates a new SessionSettingsRepository.
func NewSessionSettingsRepository(repo *dbrepo.RepoManager) SessionSettingsRepository {
	return &sessionSettingsRepository{repo: repo}
}

// selectCols lists the columns matching the domain.SessionSettings struct.
// SessionSettingsTable.SelectColumns() includes CreatedAt which the domain
// struct omits, so we list them explicitly.
var selectCols = dbrepo.Columns("Id", "SessionUUID", "Theme", "UpdatedAt")

var tableName = schema.SessionSettingsTable.Name

// GetByUUID returns settings for the given session UUID, or nil if not found.
func (r *sessionSettingsRepository) GetByUUID(ctx context.Context, uuid string) (*domain.SessionSettings, error) {
	w := dbrepo.NewWhere().And("SessionUUID = @SessionUUID", sql.Named("SessionUUID", uuid))
	query, args := dbrepo.NewSelect(tableName, selectCols).Where(w).Build()

	var s domain.SessionSettings
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
func (r *sessionSettingsRepository) Upsert(ctx context.Context, s *domain.SessionSettings) error {
	existing, err := r.GetByUUID(ctx, s.SessionUUID)
	if err != nil {
		return err
	}
	if existing != nil {
		query := fmt.Sprintf("UPDATE %s SET %s WHERE SessionUUID = @SessionUUID",
			tableName,
			dbrepo.SetClause("Theme", "UpdatedAt"),
		)
		dbrepo.SetUpdateTimestamp(&s.UpdatedAt)
		_, err = r.repo.GetDB().ExecContext(ctx, query,
			sql.Named("Theme", s.Theme),
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

// DeleteStale removes session settings rows not updated in the given number of days.
func (r *sessionSettingsRepository) DeleteStale(ctx context.Context, days int) (int64, error) {
	w := dbrepo.NewWhere().And("UpdatedAt < datetime('now', @StaleInterval)",
		sql.Named("StaleInterval", fmt.Sprintf("-%d days", days)),
	)
	query := fmt.Sprintf("DELETE FROM %s %s", tableName, w.String())
	res, err := r.repo.GetDB().ExecContext(ctx, query, w.Args()...)
	if err != nil {
		return 0, fmt.Errorf("delete stale session settings: %w", err)
	}
	return res.RowsAffected()
}
