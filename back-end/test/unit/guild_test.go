package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
)

func TestGuildHasMember(t *testing.T) {
	g := guild.Guild{ID: "g1", Name: "Família", UserIDs: []string{"lia", "beto"}}

	assert.True(t, g.HasMember("lia"))
	assert.False(t, g.HasMember("caio"))
}
