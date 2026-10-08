// Package user holds the domain model for users.
package user

import "errors"

var (
	ErrNotFound           = errors.New("user: not found")
	ErrInvalidCredentials = errors.New("user: invalid credentials")
)

// User is a person who can sign in. PasswordHash is a bcrypt hash, never the plain password.
type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
}
