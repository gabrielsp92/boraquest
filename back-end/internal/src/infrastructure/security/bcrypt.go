// Package security provides password hashing and access tokens.
package security

import "golang.org/x/crypto/bcrypt"

// BcryptComparer checks passwords against bcrypt hashes.
type BcryptComparer struct{}

// Compare returns an error unless password matches hash.
func (BcryptComparer) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
