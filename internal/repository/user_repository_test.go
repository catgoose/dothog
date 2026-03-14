// setup:feature:graph
package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"catgoose/dothog/internal/database/dialect"
	dbrepo "catgoose/dothog/internal/database/repository"
	"catgoose/dothog/internal/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	sqlxDB := sqlx.NewDb(db, "sqlite3")
	return sqlxDB, mock
}

func TestUserRepository_GetByID_Success(t *testing.T) {
	sqlxDB, mock := setupMockDB(t)
	repo := dbrepo.NewManager(sqlxDB, dialect.SQLiteDialect{})
	ur := NewUserRepository(repo)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"ID", "AzureId", "UserPrincipalName", "GivenName", "Surname", "DisplayName", "Mail", "JobTitle", "OfficeLocation", "Department", "CompanyName", "AccountName", "LastLoginAt", "CreatedAt", "UpdatedAt"}).
		AddRow(1, "azure-1", "user@example.com", "John", "Doe", "John Doe", "john@example.com", "Engineer", "Seattle", "Eng", "Acme", "jdoe", time.Now(), time.Now(), time.Now())
	mock.ExpectQuery("SELECT .* FROM Users WHERE ID").
		WithArgs(1).
		WillReturnRows(rows)

	user, err := ur.GetByID(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, 1, user.ID)
	require.Equal(t, "azure-1", user.AzureID)
	require.Equal(t, "user@example.com", user.UserPrincipalName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	sqlxDB, mock := setupMockDB(t)
	repo := dbrepo.NewManager(sqlxDB, dialect.SQLiteDialect{})
	ur := NewUserRepository(repo)
	ctx := context.Background()

	mock.ExpectQuery("SELECT .* FROM Users WHERE ID").
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	user, err := ur.GetByID(ctx, 999)
	require.Error(t, err)
	require.Nil(t, user)
	require.Contains(t, err.Error(), "user not found")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByAzureID_Success(t *testing.T) {
	sqlxDB, mock := setupMockDB(t)
	repo := dbrepo.NewManager(sqlxDB, dialect.SQLiteDialect{})
	ur := NewUserRepository(repo)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"ID", "AzureId", "UserPrincipalName", "GivenName", "Surname", "DisplayName", "Mail", "JobTitle", "OfficeLocation", "Department", "CompanyName", "AccountName", "LastLoginAt", "CreatedAt", "UpdatedAt"}).
		AddRow(1, "azure-1", "user@example.com", "John", "Doe", "John Doe", "john@example.com", "Engineer", "Seattle", "Eng", "Acme", "jdoe", time.Now(), time.Now(), time.Now())
	mock.ExpectQuery("SELECT .* FROM Users WHERE AzureId").
		WithArgs("azure-1").
		WillReturnRows(rows)

	user, err := ur.GetByAzureID(ctx, "azure-1")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, 1, user.ID)
	require.Equal(t, "azure-1", user.AzureID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByAzureID_NotFound(t *testing.T) {
	sqlxDB, mock := setupMockDB(t)
	repo := dbrepo.NewManager(sqlxDB, dialect.SQLiteDialect{})
	ur := NewUserRepository(repo)
	ctx := context.Background()

	mock.ExpectQuery("SELECT .* FROM Users WHERE AzureId").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	user, err := ur.GetByAzureID(ctx, "nonexistent")
	require.Error(t, err)
	require.Nil(t, user)
	require.Contains(t, err.Error(), "user not found")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_UpdateLastLogin_Success(t *testing.T) {
	sqlxDB, mock := setupMockDB(t)
	repo := dbrepo.NewManager(sqlxDB, dialect.SQLiteDialect{})
	ur := NewUserRepository(repo)
	ctx := context.Background()

	mock.ExpectExec("UPDATE Users").
		WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := ur.UpdateLastLogin(ctx, 1, nil)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_UpdateLastLogin_NotFound(t *testing.T) {
	sqlxDB, mock := setupMockDB(t)
	repo := dbrepo.NewManager(sqlxDB, dialect.SQLiteDialect{})
	ur := NewUserRepository(repo)
	ctx := context.Background()

	mock.ExpectExec("UPDATE Users").
		WithArgs(999, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := ur.UpdateLastLogin(ctx, 999, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "user not found")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Update_Success(t *testing.T) {
	sqlxDB, mock := setupMockDB(t)
	repo := dbrepo.NewManager(sqlxDB, dialect.SQLiteDialect{})
	ur := NewUserRepository(repo)
	ctx := context.Background()

	user := &domain.User{
		ID:                1,
		AzureID:           "azure-1",
		UserPrincipalName: "user@example.com",
		GivenName:         sql.NullString{String: "John", Valid: true},
		Surname:           sql.NullString{String: "Doe", Valid: true},
		DisplayName:       sql.NullString{String: "John Doe", Valid: true},
		Mail:              sql.NullString{String: "john@example.com", Valid: true},
		JobTitle:          sql.NullString{String: "Engineer", Valid: true},
		OfficeLocation:    sql.NullString{String: "Seattle", Valid: true},
		Department:        sql.NullString{String: "Eng", Valid: true},
		CompanyName:       sql.NullString{String: "Acme", Valid: true},
		AccountName:       sql.NullString{String: "jdoe", Valid: true},
		LastLoginAt:       sql.NullTime{Time: time.Now(), Valid: true},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	mock.ExpectExec("UPDATE Users").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := ur.Update(ctx, user, nil)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
