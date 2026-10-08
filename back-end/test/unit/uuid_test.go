package unit_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/idgen"
)

func TestUUIDGenerator(t *testing.T) {
	gen := idgen.UUIDGenerator{}

	a, b := gen.NewID(), gen.NewID()

	parsed, err := uuid.Parse(a)
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(4), parsed.Version())
	assert.NotEqual(t, a, b)
}
