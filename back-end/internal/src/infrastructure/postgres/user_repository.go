package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

// PostgreSQL error codes and the constraint names (from the migrations) Create maps to domain errors.
const (
	uniqueViolation       = "23505"
	foreignKeyViolation   = "23503"
	usersEmailKey         = "users_email_key"
	guildMembersGuildFKey = "guild_members_guild_id_fkey"
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

// Create inserts u as a member of guildID, atomically. A taken email yields
// user.ErrAlreadyExists and an unknown guild guild.ErrNotFound.
func (r *UserRepository) Create(ctx context.Context, u user.User, guildID string) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO users (id, name, email, password_hash) VALUES ($1, $2, $3, $4)`,
			u.ID, u.Name, u.Email, u.PasswordHash)
		if isConstraintError(err, uniqueViolation, usersEmailKey) {
			return user.ErrAlreadyExists
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO guild_members (user_id, guild_id) VALUES ($1, $2)`, u.ID, guildID)
		if isConstraintError(err, foreignKeyViolation, guildMembersGuildFKey) {
			return guild.ErrNotFound
		}
		return err
	})
}

// isConstraintError reports whether err is a PostgreSQL error with that code on that constraint.
func isConstraintError(err error, code, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code && pgErr.ConstraintName == constraint
}
