package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

const (
	// uniqueViolation is the PostgreSQL error code for a unique constraint failure.
	uniqueViolation = "23505"
	// usersEmailKey is the unique constraint on users.email (0001_users.sql).
	usersEmailKey = "users_email_key"
)

// UserRepository reads and writes users in the users table.
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

// Create inserts u, mapping a duplicate email to user.ErrAlreadyExists.
func (r *UserRepository) Create(ctx context.Context, u user.User) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO users (id, name, email, password_hash) VALUES ($1, $2, $3, $4)`,
		u.ID, u.Name, u.Email, u.PasswordHash)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation && pgErr.ConstraintName == usersEmailKey {
		return user.ErrAlreadyExists
	}
	return err
}
