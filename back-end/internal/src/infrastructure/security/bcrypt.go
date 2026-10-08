// Package security provides password hashing and access tokens.
package security

import "golang.org/x/crypto/bcrypt"

// BcryptComparer checks passwords against bcrypt hashes.
type BcryptComparer struct{}

// Compare returns an error unless password matches hash.
func (BcryptComparer) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// BcryptHasher hashes passwords with bcrypt. A zero Cost uses bcrypt.DefaultCost.
type BcryptHasher struct {
	Cost int
}

// Hash returns the bcrypt hash of password.
func (h BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.Cost)
	return string(hash), err
}
