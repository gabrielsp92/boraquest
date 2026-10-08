package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

// UserRepository reads users from the users table.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository builds a UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// FindByEmail returns the user with that email.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (user.User, error) {
	var u user.User
	err := r.pool.QueryRow(ctx, `SELECT id, name, email, password_hash FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, user.ErrNotFound
	}
	return u, err
}
