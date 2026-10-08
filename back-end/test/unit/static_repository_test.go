package unit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	guildrepo "github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/guild"
)

func TestStaticRepositoryFindByMember(t *testing.T) {
	other := guild.Guild{ID: "g2", UserIDs: []string{"tom"}}
	repo := guildrepo.NewStaticRepository(guildrepo.Default, other)

	g, err := repo.FindByMember(context.Background(), "nena")
	require.NoError(t, err)
	assert.Equal(t, guildrepo.Default, g)

	g, err = repo.FindByMember(context.Background(), "tom")
	require.NoError(t, err)
	assert.Equal(t, "g2", g.ID)

	_, err = repo.FindByMember(context.Background(), "intruder")
	assert.ErrorIs(t, err, guild.ErrNotMember)
}

func TestDefaultGuildMembers(t *testing.T) {
	assert.Equal(t, "familia", guildrepo.Default.ID)
	assert.ElementsMatch(t, []string{"lia", "beto", "nena", "caio"}, guildrepo.Default.UserIDs)
}
