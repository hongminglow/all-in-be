package storage

import (
	"context"
	"errors"

	"github.com/hongminglow/all-in-be/internal/models"
)

// ErrNotFound indicates a record does not exist.
var ErrNotFound = errors.New("record not found")

// ErrAlreadyExists indicates a uniqueness conflict.
var ErrAlreadyExists = errors.New("record already exists")

// UserStore captures persistence operations needed by handlers.
type UserStore interface {
	CreateUser(ctx context.Context, user models.User) (models.User, error)
	FindByUsername(ctx context.Context, username string) (models.User, error)
	FindByEmail(ctx context.Context, email string) (models.User, error)
	FindByUsernameOrEmail(ctx context.Context, identifier string) (models.User, error)
}
