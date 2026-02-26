// setup:feature:graph

package repository

import (
	"context"

	"catgoose/go-htmx-template/internals/domain"

	"github.com/jmoiron/sqlx"
)

// UserRepository defines operations for user data access
type UserRepository interface {
	CreateOrUpdate(ctx context.Context, user *domain.User, tx *sqlx.Tx) error
	GetByID(ctx context.Context, id int) (*domain.User, error)
	GetByAzureID(ctx context.Context, azureID string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User, tx *sqlx.Tx) error
	UpdateLastLogin(ctx context.Context, id int, tx *sqlx.Tx) error
}
