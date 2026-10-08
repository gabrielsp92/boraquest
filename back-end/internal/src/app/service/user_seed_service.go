package service

import (
	"context"
	"errors"
	"strings"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

// UserSeedRepository looks up and stores users.
type UserSeedRepository interface {
	// FindByEmail returns user.ErrNotFound when no user has that email.
	FindByEmail(ctx context.Context, email string) (user.User, error)
	// Create stores u as a member of guildID, atomically. It returns
	// user.ErrAlreadyExists when the email is taken and guild.ErrNotFound when
	// the guild does not exist.
	Create(ctx context.Context, u user.User, guildID string) error
}

// GuildFinder looks up guilds by id.
type GuildFinder interface {
	// Get returns guild.ErrNotFound when no guild has that id.
	Get(ctx context.Context, id string) (guild.Guild, error)
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

// SeedUserInput describes the user to seed and the guild (by id) it joins.
// DryRun validates and checks the database without writing anything.
type SeedUserInput struct {
	Name     string
	Email    string
	Password string
	GuildID  string
	DryRun   bool
}

// SeedUserResult is the seeded user (never with a password hash in a dry run),
// the guild it joins and what happened to it. Guild is left empty when the
// user already existed.
type SeedUserResult struct {
	User    user.User
	Guild   guild.Guild
	Outcome SeedOutcome
}

// UserSeedService adds users outside the sign-up flow, e.g. from a script.
type UserSeedService struct {
	users     UserSeedRepository
	guilds    GuildFinder
	passwords PasswordHasher
	ids       IDGenerator
}

// NewUserSeedService builds a UserSeedService.
func NewUserSeedService(users UserSeedRepository, guilds GuildFinder, passwords PasswordHasher, ids IDGenerator) *UserSeedService {
	return &UserSeedService{users: users, guilds: guilds, passwords: passwords, ids: ids}
}

// Seed creates the user as a member of the input guild, which must exist.
// When a user with the same email already exists it is returned untouched
// (neither its password nor its guild changes).
func (s *UserSeedService) Seed(ctx context.Context, in SeedUserInput) (SeedUserResult, error) {
	if err := user.ValidatePassword(in.Password); err != nil {
		return SeedUserResult{}, err
	}
	u, err := user.New(s.ids.NewID(), in.Name, in.Email, "")
	if err != nil {
		return SeedUserResult{}, err
	}
	g, err := s.guilds.Get(ctx, strings.TrimSpace(in.GuildID))
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
		return SeedUserResult{User: u, Guild: g, Outcome: SeedWouldCreate}, nil
	}
	if u.PasswordHash, err = s.passwords.Hash(in.Password); err != nil {
		return SeedUserResult{}, err
	}
	if err := s.users.Create(ctx, u, g.ID); err != nil {
		return SeedUserResult{}, err
	}
	return SeedUserResult{User: u, Guild: g, Outcome: SeedCreated}, nil
}
