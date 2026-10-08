package unit_test

import (
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
