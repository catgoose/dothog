// setup:feature:graph

package repository

import (
	"context"

	"catgoose/dothog/internal/domain"

	"github.com/jmoiron/sqlx"
)

// UserRepository is the persistence contract for the Users table; every
// method accepts a *sqlx.Tx and may be nil to run on the underlying DB.
type UserRepository interface {
	CreateOrUpdate(ctx context.Context, user *domain.User, tx *sqlx.Tx) error
	GetByID(ctx context.Context, id int) (*domain.User, error)
	GetByAzureID(ctx context.Context, azureID string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User, tx *sqlx.Tx) error
	UpdateLastLogin(ctx context.Context, id int, tx *sqlx.Tx) error
}
