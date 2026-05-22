// setup:feature:graph

package graph

import (
	"context"
	"fmt"
	"strings"

	"database/sql"

	"catgoose/dothog/internal/domain"
	"catgoose/dothog/internal/logger"

	"github.com/jmoiron/sqlx"
)

// UserCache is the SQLite-backed cache of Graph users (Users table).
type UserCache struct {
	DB *sqlx.DB
}

// NewUserCache binds db without verifying the Users table exists; call EnsureSchema or
// UsersTableExists when that guarantee is needed.
func NewUserCache(db *sqlx.DB) *UserCache {
	return &UserCache{DB: db}
}

// InsertOrUpdateUsers upserts each user in a single transaction, rolling back on the first row that fails.
func (c *UserCache) InsertOrUpdateUsers(ctx context.Context, users []domain.GraphUser) error {
	log := logger.WithContext(ctx)

	if err := c.EnsureSchema(ctx); err != nil {
		return fmt.Errorf("failed to ensure schema: %w", err)
	}

	tx, err := c.DB.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Error("Failed to rollback transaction", "rollback_error", rollbackErr, "original_error", err)
			}
		}
	}()

	query := `
		INSERT INTO Users (AzureId, GivenName, Surname, DisplayName, UserPrincipalName, Mail, JobTitle, OfficeLocation, Department, CompanyName, AccountName, UpdatedAt)
		VALUES (:AzureId, :GivenName, :Surname, :DisplayName, :UserPrincipalName, :Mail, :JobTitle, :OfficeLocation, :Department, :CompanyName, :AccountName, CURRENT_TIMESTAMP)
		ON CONFLICT(AzureId) DO UPDATE SET
		  AzureId = excluded.AzureId
			, GivenName = excluded.GivenName
			, Surname = excluded.Surname
			, DisplayName = excluded.DisplayName
			, UserPrincipalName = excluded.UserPrincipalName
			, Mail = excluded.Mail
			, JobTitle = excluded.JobTitle
			, OfficeLocation = excluded.OfficeLocation
			, Department = excluded.Department
			, CompanyName = excluded.CompanyName
			, AccountName = excluded.AccountName
			, UpdatedAt = CURRENT_TIMESTAMP;
	`

	for _, user := range users {
		if _, err = tx.NamedExec(query, user); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Error("Failed to rollback transaction after insert/update error", "rollback_error", rollbackErr, "original_error", err, "user_id", user.AzureID)
			}
			return fmt.Errorf("failed to insert/update user %s: %w", user.AzureID, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			log.Error("Failed to rollback transaction after commit error", "rollback_error", rollbackErr, "commit_error", err)
		}
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

const (
	userSelect = "SELECT AzureId, GivenName, Surname, DisplayName, UserPrincipalName, Mail, JobTitle, OfficeLocation, Department, CompanyName, AccountName"
)

// SearchUsers ANDs LIKE clauses across GivenName/Surname/DisplayName/AccountName for each term and returns up to limit rows ordered by DisplayName.
func (c *UserCache) SearchUsers(terms []string, limit int) ([]domain.GraphUser, error) {
	if len(terms) == 0 {
		return nil, fmt.Errorf("no search terms provided")
	}
	var conditions []string
	var args []any

	for i, term := range terms {
		searchPattern := "%" + term + "%"
		paramName := fmt.Sprintf("Search%d", i)
		conditions = append(conditions, fmt.Sprintf("(GivenName LIKE @%s OR Surname LIKE @%s OR DisplayName LIKE @%s OR AccountName LIKE @%s)", paramName, paramName, paramName, paramName))
		args = append(args, sql.Named(paramName, searchPattern))
	}
	whereClause := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`
	  %s
		FROM Users
		WHERE (%s)
		ORDER BY DisplayName
		LIMIT @Limit
	`, userSelect, whereClause)

	args = append(args, sql.Named("Limit", limit))

	var users []domain.GraphUser
	err := c.DB.Select(&users, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	return users, nil
}

// GetAllUsers dumps the entire Users table (DisplayName-sorted). Intended
// for one-off admin views; large directories should prefer SearchUsers.
func (c *UserCache) GetAllUsers() ([]domain.GraphUser, error) {
	query := `
		SELECT AzureId, GivenName, Surname, DisplayName, UserPrincipalName, Mail, JobTitle, OfficeLocation, Department, CompanyName, AccountName
		FROM Users
		ORDER BY DisplayName
	`
	var users []domain.GraphUser
	err := c.DB.Select(&users, query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve all users: %w", err)
	}
	return users, nil
}

// GetUserByAzureID wraps sql.ErrNoRows in a contextual error when the row is missing.
func (c *UserCache) GetUserByAzureID(azureID string) (*domain.GraphUser, error) {
	query := `
		SELECT AzureId, GivenName, Surname, DisplayName, UserPrincipalName, Mail, JobTitle, OfficeLocation, Department, CompanyName, AccountName
		FROM Users
		WHERE AzureId = @AzureId
	`
	var user domain.GraphUser
	err := c.DB.Get(&user, query, sql.Named("AzureId", azureID))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user by Azure ID: %w", err)
	}
	return &user, nil
}

// UsersTableExists probes sqlite_master directly so callers can decide whether
// to call EnsureSchema before queries.
func (c *UserCache) UsersTableExists() (bool, error) {
	var count int
	err := c.DB.Get(&count, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='Users'")
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// EnsureSchema is idempotent: creates the Users table only when sqlite_master reports it missing.
func (c *UserCache) EnsureSchema(ctx context.Context) error {
	exists, err := c.UsersTableExists()
	if err != nil {
		return fmt.Errorf("failed to check if Users table exists: %w", err)
	}

	if !exists {
		log := logger.WithContext(ctx)
		log.Info("Users table does not exist, creating it")

		usersTableSchema := `
		CREATE TABLE IF NOT EXISTS Users (
			AzureId TEXT PRIMARY KEY,
			GivenName TEXT,
			Surname TEXT,
			DisplayName TEXT,
			UserPrincipalName TEXT,
			Mail TEXT,
			JobTitle TEXT,
			OfficeLocation TEXT,
			Department TEXT,
			CompanyName TEXT,
			AccountName TEXT,
			UpdatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`

		_, err = c.DB.Exec(usersTableSchema)
		if err != nil {
			return fmt.Errorf("failed to create Users table: %w", err)
		}

		log.Info("Users table created successfully")
	}

	return nil
}

// GetUserCount is the row count of Users (cached and persisted).
func (c *UserCache) GetUserCount() (int, error) {
	var count int
	err := c.DB.Get(&count, "SELECT COUNT(*) FROM Users")
	if err != nil {
		return 0, fmt.Errorf("failed to get user count: %w", err)
	}
	return count, nil
}
