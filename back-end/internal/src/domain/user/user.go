// Package user holds the domain model for users.
package user

import (
	"errors"
	"net/mail"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	MaxNameLen = 64
	// MinPasswordLen and MaxPasswordLen are in bytes; bcrypt rejects passwords over 72 bytes.
	MinPasswordLen = 8
	MaxPasswordLen = 72
)

var (
	ErrNotFound           = errors.New("user: not found")
	ErrInvalidCredentials = errors.New("user: invalid credentials")
	ErrAlreadyExists      = errors.New("user: email already in use")
	ErrInvalidName        = errors.New("user: name must have 1 to 64 characters")
	ErrInvalidEmail       = errors.New("user: invalid email")
	ErrInvalidPassword    = errors.New("user: password must have 8 to 72 bytes")
	ErrInvalidAvatar      = errors.New("user: invalid avatar")
)

// AvatarKeys are the platform's known avatar icons.
var AvatarKeys = []string{"beto", "caio", "duda", "lia", "nena", "tom"}

// ValidAvatarKey reports whether key is one of the platform's known avatars.
func ValidAvatarKey(key string) bool {
	return slices.Contains(AvatarKeys, key)
}

// User is a person who can sign in. PasswordHash is a bcrypt hash, never the plain password.
type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	AvatarKey    string
}

// New builds a validated User. The name is trimmed and the email normalized.
func New(id, name, email, passwordHash string) (User, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n == 0 || n > MaxNameLen {
		return User{}, ErrInvalidName
	}
	email = NormalizeEmail(email)
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return User{}, ErrInvalidEmail
	}
	return User{ID: id, Name: name, Email: email, PasswordHash: passwordHash}, nil
}

// NormalizeEmail returns the canonical form emails are stored and looked up in.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidatePassword reports whether password is acceptable for a new user.
func ValidatePassword(password string) error {
	if n := len(password); n < MinPasswordLen || n > MaxPasswordLen {
		return ErrInvalidPassword
	}
	return nil
}
