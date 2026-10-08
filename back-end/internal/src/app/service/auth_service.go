package service

import (
	"context"
	"errors"
	"strings"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

// UserRepository looks up users.
type UserRepository interface {
	// FindByEmail returns user.ErrNotFound when no user has that email.
	FindByEmail(ctx context.Context, email string) (user.User, error)
}

// PasswordComparer checks a plain password against a stored hash.
type PasswordComparer interface {
	// Compare returns a non-nil error when password does not match hash.
	Compare(hash, password string) error
}

// TokenIssuer issues access tokens that identify a user.
type TokenIssuer interface {
	Issue(userID string) (string, error)
}

// dummyHash is compared against when the email is unknown, so a login attempt
// takes the same time whether or not the user exists.
const dummyHash = "$2a$10$0TUeb.eJrmCxiZbvigDvXOQ.yL2F2zzpFe.cHKn5CKflM0fsjOPNC"

// AuthService signs users in.
type AuthService struct {
	users     UserRepository
	passwords PasswordComparer
	tokens    TokenIssuer
}

// NewAuthService builds an AuthService.
func NewAuthService(users UserRepository, passwords PasswordComparer, tokens TokenIssuer) *AuthService {
	return &AuthService{users: users, passwords: passwords, tokens: tokens}
}

// Login checks the credentials and returns an access token for the user.
// Unknown emails and wrong passwords both yield user.ErrInvalidCredentials.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, user.User, error) {
	u, err := s.users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, user.ErrNotFound) {
		_ = s.passwords.Compare(dummyHash, password)
		return "", user.User{}, user.ErrInvalidCredentials
	}
	if err != nil {
		return "", user.User{}, err
	}
	if err := s.passwords.Compare(u.PasswordHash, password); err != nil {
		return "", user.User{}, user.ErrInvalidCredentials
	}
	token, err := s.tokens.Issue(u.ID)
	if err != nil {
		return "", user.User{}, err
	}
	return token, u, nil
}
