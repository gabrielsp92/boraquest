package unit_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/prize"
)

func TestPrizeClean(t *testing.T) {
	clean, err := prize.Clean("  Jantar no restaurante favorito  ")

	require.NoError(t, err)
	assert.Equal(t, "Jantar no restaurante favorito", clean)
}

func TestPrizeCleanEmptyIsValid(t *testing.T) {
	clean, err := prize.Clean("   ")

	require.NoError(t, err)
	assert.Equal(t, "", clean)
}

func TestPrizeCleanMaxLenIsValid(t *testing.T) {
	text := strings.Repeat("á", prize.MaxLen)

	clean, err := prize.Clean(text)

	require.NoError(t, err)
	assert.Equal(t, text, clean)
}

func TestPrizeCleanTooLong(t *testing.T) {
	text := strings.Repeat("á", prize.MaxLen+1)

	_, err := prize.Clean(text)

	assert.ErrorIs(t, err, prize.ErrTooLong)
}
