package unit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

type fakeGuildUserRepo struct {
	users []user.User
	err   error

	gotIDs []string
}

func (f *fakeGuildUserRepo) ListByIDs(_ context.Context, ids []string) ([]user.User, error) {
	f.gotIDs = ids
	return f.users, f.err
}

func newGuildService(guilds fakeGuilds, users *fakeGuildUserRepo) *service.GuildService {
	return service.NewGuildService(guilds, users)
}

func TestGuildServiceListMembers(t *testing.T) {
	g := guild.Guild{ID: "g1", Name: "Família", UserIDs: []string{"lia", "beto"}}
	users := &fakeGuildUserRepo{users: []user.User{
		{ID: "beto", Name: "Beto"},
		{ID: "lia", Name: "Lia"},
	}}

	members, err := newGuildService(fakeGuilds{guild: g}, users).ListMembers(context.Background(), "lia")

	require.NoError(t, err)
	assert.Equal(t, []service.GuildMember{{ID: "lia", Name: "Lia"}, {ID: "beto", Name: "Beto"}}, members, "order follows the guild's UserIDs, not the repo's return order")
	assert.Equal(t, []string{"lia", "beto"}, users.gotIDs)
}

func TestGuildServiceListMembersRejectsNonMember(t *testing.T) {
	users := &fakeGuildUserRepo{}

	_, err := newGuildService(fakeGuilds{err: guild.ErrNotMember}, users).ListMembers(context.Background(), "x")

	assert.ErrorIs(t, err, guild.ErrNotMember)
	assert.Nil(t, users.gotIDs)
}

func TestGuildServiceListMembersRepoError(t *testing.T) {
	g := guild.Guild{ID: "g1", Name: "Família", UserIDs: []string{"lia"}}
	users := &fakeGuildUserRepo{err: errBoom}

	_, err := newGuildService(fakeGuilds{guild: g}, users).ListMembers(context.Background(), "lia")

	assert.ErrorIs(t, err, errBoom)
}
