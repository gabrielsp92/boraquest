package unit_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/security"
)

func TestBcryptComparer(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	require.NoError(t, err)

	assert.NoError(t, security.BcryptComparer{}.Compare(string(hash), "secret"))
	assert.Error(t, security.BcryptComparer{}.Compare(string(hash), "wrong"))
	assert.Error(t, security.BcryptComparer{}.Compare("not-a-hash", "secret"))
}

func TestBcryptHasher(t *testing.T) {
	hasher := security.BcryptHasher{Cost: bcrypt.MinCost}

	hash, err := hasher.Hash("secret")
	require.NoError(t, err)
	assert.NoError(t, security.BcryptComparer{}.Compare(hash, "secret"))

	_, err = hasher.Hash(strings.Repeat("x", 73))
	assert.ErrorIs(t, err, bcrypt.ErrPasswordTooLong)
}
