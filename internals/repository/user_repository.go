// setup:feature:graph

package repository

import (
	"context"
	"database/sql"
	"fmt"

	"catgoose/go-htmx-demo/internals/database/repository"
	"catgoose/go-htmx-demo/internals/database/schema"
	"catgoose/go-htmx-demo/internals/domain"

	"github.com/jmoiron/sqlx"
)

// userRepository implements UserRepository
type userRepository struct {
	repo *repository.RepoManager
}

// NewUserRepository creates a new UserRepository instance
func NewUserRepository(repo *repository.RepoManager) UserRepository {
	return &userRepository{repo: repo}
}

// CreateOrUpdate creates a new user or updates an existing one based on AzureID
func (r *userRepository) CreateOrUpdate(ctx context.Context, user *domain.User, tx *sqlx.Tx) error {
	existing, err := r.getByAzureIDInternal(ctx, user.AzureID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check for existing user: %w", err)
	}

	now := repository.GetNow()

	if existing != nil {
		user.ID = existing.ID
		user.CreatedAt = existing.CreatedAt
		user.LastLoginAt = sql.NullTime{Time: now, Valid: true}
		repository.SetUpdateTimestamp(&user.UpdatedAt)
		return r.Update(ctx, user, tx)
	}

	repository.SetCreateTimestamps(&user.CreatedAt, &user.UpdatedAt)
	user.LastLoginAt = sql.NullTime{Time: now, Valid: true}

	var exec repository.GetExecer = r.repo.GetDB()
	if tx != nil {
		exec = tx
	}

	insertCols := schema.UsersTable.InsertColumns()
	query := repository.InsertInto(schema.UsersTable.Name, insertCols...) + ";\n\t\tSELECT SCOPE_IDENTITY() AS ID;"

	var id int64
	err = exec.GetContext(ctx, &id, query,
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

// GetByID retrieves a user by its ID
func (r *userRepository) GetByID(ctx context.Context, id int) (*domain.User, error) {
	query := `SELECT * FROM Users WHERE ID = @ID`
	var user domain.User
	err := r.repo.GetDB().GetContext(ctx, &user, query, sql.Named("ID", id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// getByAzureIDInternal retrieves a user by their Azure ID (internal method that returns sql.ErrNoRows)
func (r *userRepository) getByAzureIDInternal(ctx context.Context, azureID string) (*domain.User, error) {
	query := `SELECT * FROM Users WHERE AzureId = @AzureId`
	var user domain.User
	err := r.repo.GetDB().GetContext(ctx, &user, query, sql.Named("AzureId", azureID))
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByAzureID retrieves a user by their Azure ID
func (r *userRepository) GetByAzureID(ctx context.Context, azureID string) (*domain.User, error) {
	user, err := r.getByAzureIDInternal(ctx, azureID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user by Azure ID: %w", err)
	}
	return user, nil
}

// Update updates an existing user
func (r *userRepository) Update(ctx context.Context, user *domain.User, tx *sqlx.Tx) error {
	var exec repository.GetExecer = r.repo.GetDB()
	if tx != nil {
		exec = tx
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE ID = @ID",
		schema.UsersTable.Name,
		repository.SetClause(schema.UsersTable.UpdateColumns()...),
	)

	repository.SetUpdateTimestamp(&user.UpdatedAt)

	result, err := exec.ExecContext(ctx, query,
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
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdateLastLogin updates only the LastLoginAt timestamp for a user
func (r *userRepository) UpdateLastLogin(ctx context.Context, id int, tx *sqlx.Tx) error {
	var exec repository.GetExecer = r.repo.GetDB()
	if tx != nil {
		exec = tx
	}

	query := `
		UPDATE Users
		SET LastLoginAt = @LastLoginAt,
		    UpdatedAt = @UpdatedAt
		WHERE ID = @ID
	`

	now := repository.GetNow()
	result, err := exec.ExecContext(ctx, query,
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
		return fmt.Errorf("user not found")
	}

	return nil
}
