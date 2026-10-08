package service

import (
	"context"
	"errors"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

// UserSeedRepository looks up and stores users.
type UserSeedRepository interface {
	// FindByEmail returns user.ErrNotFound when no user has that email.
	FindByEmail(ctx context.Context, email string) (user.User, error)
	// Create returns user.ErrAlreadyExists when the email is already taken.
	Create(ctx context.Context, u user.User) error
}

// PasswordHasher turns a plain password into a storable hash.
type PasswordHasher interface {
	Hash(password string) (string, error)
}

// SeedOutcome says what Seed did (or, in a dry run, would do).
type SeedOutcome string

const (
	SeedCreated       SeedOutcome = "created"
	SeedWouldCreate   SeedOutcome = "would-create"
	SeedAlreadyExists SeedOutcome = "already-exists"
)

// SeedUserInput describes the user to seed. DryRun validates and checks the
// database without writing anything.
type SeedUserInput struct {
	Name     string
	Email    string
	Password string
	DryRun   bool
}

// SeedUserResult is the seeded user (never with a password hash in a dry run)
// and what happened to it.
type SeedUserResult struct {
	User    user.User
	Outcome SeedOutcome
}

// UserSeedService adds users outside the sign-up flow, e.g. from a script.
type UserSeedService struct {
	users     UserSeedRepository
	passwords PasswordHasher
	ids       IDGenerator
}

// NewUserSeedService builds a UserSeedService.
func NewUserSeedService(users UserSeedRepository, passwords PasswordHasher, ids IDGenerator) *UserSeedService {
	return &UserSeedService{users: users, passwords: passwords, ids: ids}
}

// Seed creates the user unless one with the same email exists, in which case
// the existing user is returned untouched (its password is not changed).
func (s *UserSeedService) Seed(ctx context.Context, in SeedUserInput) (SeedUserResult, error) {
	if err := user.ValidatePassword(in.Password); err != nil {
		return SeedUserResult{}, err
	}
	u, err := user.New(s.ids.NewID(), in.Name, in.Email, "")
	if err != nil {
		return SeedUserResult{}, err
	}
	existing, err := s.users.FindByEmail(ctx, u.Email)
	if err == nil {
		return SeedUserResult{User: existing, Outcome: SeedAlreadyExists}, nil
	}
	if !errors.Is(err, user.ErrNotFound) {
		return SeedUserResult{}, err
	}
	if in.DryRun {
		return SeedUserResult{User: u, Outcome: SeedWouldCreate}, nil
	}
	if u.PasswordHash, err = s.passwords.Hash(in.Password); err != nil {
		return SeedUserResult{}, err
	}
	if err := s.users.Create(ctx, u); err != nil {
		return SeedUserResult{}, err
	}
	return SeedUserResult{User: u, Outcome: SeedCreated}, nil
}
