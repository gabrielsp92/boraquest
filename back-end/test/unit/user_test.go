package unit_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

func TestNewUser(t *testing.T) {
	u, err := user.New("u1", "  Vó Nena  ", "  Nena@BoraQuest.dev ", "hash")

	require.NoError(t, err)
	assert.Equal(t, user.User{ID: "u1", Name: "Vó Nena", Email: "nena@boraquest.dev", PasswordHash: "hash"}, u)
}

func TestNewUserValidation(t *testing.T) {
	cases := map[string]struct {
		name, email string
		want        error
	}{
		"blank name":         {"   ", "a@b.dev", user.ErrInvalidName},
		"name too long":      {strings.Repeat("é", user.MaxNameLen+1), "a@b.dev", user.ErrInvalidName},
		"blank email":        {"Ana", "  ", user.ErrInvalidEmail},
		"no at sign":         {"Ana", "ana.boraquest.dev", user.ErrInvalidEmail},
		"display-name email": {"Ana", "Ana <ana@b.dev>", user.ErrInvalidEmail},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := user.New("u1", tc.name, tc.email, "")
			assert.ErrorIs(t, err, tc.want)
		})
	}

	_, err := user.New("u1", strings.Repeat("é", user.MaxNameLen), "a@b.dev", "")
	assert.NoError(t, err, "max-length name counts runes, not bytes")
}

func TestNormalizeEmail(t *testing.T) {
	assert.Equal(t, "lia@boraquest.dev", user.NormalizeEmail("  LIA@Boraquest.dev\n"))
}

func TestValidatePassword(t *testing.T) {
	assert.NoError(t, user.ValidatePassword(strings.Repeat("x", user.MinPasswordLen)))
	assert.NoError(t, user.ValidatePassword(strings.Repeat("x", user.MaxPasswordLen)))
	assert.ErrorIs(t, user.ValidatePassword(strings.Repeat("x", user.MinPasswordLen-1)), user.ErrInvalidPassword)
	assert.ErrorIs(t, user.ValidatePassword(strings.Repeat("x", user.MaxPasswordLen+1)), user.ErrInvalidPassword)
}
