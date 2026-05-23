// setup:feature:graph

package graph

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/catgoose/chuck/dbrepo"
	"github.com/jmoiron/sqlx"
)

// ErrUserNotFound is returned when an Update/UpdateLastLogin affects zero rows.
var ErrUserNotFound = fmt.Errorf("user not found")

// UserRepository persists rows in the Users table — the chuck-backed mirror
// of the Graph user directory. tx-accepting methods run on the underlying
// DB when tx is nil. Lives in the graph package because both the model
// (User, GraphUser) and the schema (UsersTable) are graph-owned.
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository binds db; callers depend on *UserRepository directly
// because the package has only one Users-table persistence implementation.
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) getContext(ctx context.Context, tx *sqlx.Tx, dest any, query string, args ...any) error {
	if tx != nil {
		return tx.GetContext(ctx, dest, query, args...)
	}
	return r.db.GetContext(ctx, dest, query, args...)
}

func (r *UserRepository) execContext(ctx context.Context, tx *sqlx.Tx, query string, args ...any) (sql.Result, error) {
	if tx != nil {
		return tx.ExecContext(ctx, query, args...)
	}
	return r.db.ExecContext(ctx, query, args...)
}

// CreateOrUpdate upserts user keyed by AzureID, stamping LastLoginAt to now.
func (r *UserRepository) CreateOrUpdate(ctx context.Context, user *User, tx *sqlx.Tx) error {
	existing, err := r.getByAzureIDInternal(ctx, user.AzureID, tx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check for existing user: %w", err)
	}

	now := dbrepo.GetNow()

	if existing != nil {
		user.ID = existing.ID
		user.CreatedAt = existing.CreatedAt
		user.LastLoginAt = sql.NullTime{Time: now, Valid: true}
		dbrepo.SetUpdateTimestamp(&user.UpdatedAt)
		return r.Update(ctx, user, tx)
	}

	dbrepo.SetCreateTimestamps(&user.CreatedAt, &user.UpdatedAt)
	user.LastLoginAt = sql.NullTime{Time: now, Valid: true}

	insertCols := UsersTable.InsertColumns()
	query := dbrepo.InsertInto(UsersTable.Name, insertCols...) + ";\n\t\tSELECT SCOPE_IDENTITY() AS ID;"

	var id int64
	err = r.getContext(ctx, tx, &id, query,
		sql.Named("AzureId", user.AzureID),
		sql.Named("GivenName", user.GivenName),
		sql.Named("Surname", user.Surname),
		sql.Named("DisplayName", user.DisplayName),
		sql.Named("UserPrincipalName", user.UserPrincipalName),
		sql.Named("Mail", user.Mail),
		sql.Named("JobTitle", user.JobTitle),
		sql.Named("OfficeLocation", user.OfficeLocation),
		sql.Named("Department", user.Department),
		sql.Named("CompanyName", user.CompanyName),
		sql.Named("AccountName", user.AccountName),
		sql.Named("LastLoginAt", user.LastLoginAt),
		sql.Named("CreatedAt", user.CreatedAt),
		sql.Named("UpdatedAt", user.UpdatedAt),
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	user.ID = int(id)
	return nil
}

// GetByID returns the row matching id; sql.ErrNoRows becomes a wrapped "user not found".
func (r *UserRepository) GetByID(ctx context.Context, id int) (*User, error) {
	cols := dbrepo.Columns(UsersTable.SelectColumns()...)
	w := dbrepo.NewWhere().And("ID = @ID", sql.Named("ID", id))
	query, args := dbrepo.NewSelect(UsersTable.Name, cols).Where(w).Build()

	var user User
	err := r.db.GetContext(ctx, &user, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// getByAzureIDInternal surfaces sql.ErrNoRows so callers can distinguish "missing" from other errors.
func (r *UserRepository) getByAzureIDInternal(ctx context.Context, azureID string, tx *sqlx.Tx) (*User, error) {
	cols := dbrepo.Columns(UsersTable.SelectColumns()...)
	w := dbrepo.NewWhere().And("AzureId = @AzureId", sql.Named("AzureId", azureID))
	query, args := dbrepo.NewSelect(UsersTable.Name, cols).Where(w).Build()

	var user User
	err := r.getContext(ctx, tx, &user, query, args...)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByAzureID returns the row matching azureID; sql.ErrNoRows becomes a wrapped "user not found".
func (r *UserRepository) GetByAzureID(ctx context.Context, azureID string) (*User, error) {
	user, err := r.getByAzureIDInternal(ctx, azureID, nil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user by Azure ID: %w", err)
	}
	return user, nil
}

// Update yields ErrUserNotFound when no row matches user.ID.
func (r *UserRepository) Update(ctx context.Context, user *User, tx *sqlx.Tx) error {
	query := fmt.Sprintf("UPDATE %s SET %s WHERE ID = @ID",
		UsersTable.Name,
		dbrepo.SetClause(UsersTable.UpdateColumns()...),
	)

	dbrepo.SetUpdateTimestamp(&user.UpdatedAt)

	result, err := r.execContext(ctx, tx, query,
		sql.Named("ID", user.ID),
		sql.Named("AzureId", user.AzureID),
		sql.Named("GivenName", user.GivenName),
		sql.Named("Surname", user.Surname),
		sql.Named("DisplayName", user.DisplayName),
		sql.Named("UserPrincipalName", user.UserPrincipalName),
		sql.Named("Mail", user.Mail),
		sql.Named("JobTitle", user.JobTitle),
		sql.Named("OfficeLocation", user.OfficeLocation),
		sql.Named("Department", user.Department),
		sql.Named("CompanyName", user.CompanyName),
		sql.Named("AccountName", user.AccountName),
		sql.Named("LastLoginAt", user.LastLoginAt),
		sql.Named("UpdatedAt", user.UpdatedAt),
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateLastLogin yields ErrUserNotFound when no row matches id.
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id int, tx *sqlx.Tx) error {
	query := fmt.Sprintf("UPDATE %s SET %s WHERE ID = @ID",
		UsersTable.Name,
		dbrepo.SetClause("LastLoginAt", "UpdatedAt"),
	)

	now := dbrepo.GetNow()
	result, err := r.execContext(ctx, tx, query,
		sql.Named("ID", id),
		sql.Named("LastLoginAt", now),
		sql.Named("UpdatedAt", now),
	)

	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}
