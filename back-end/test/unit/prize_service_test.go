package unit_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/prize"
)

type fakePrizeRepo struct {
	got    prize.Prizes
	getErr error

	upserted prize.Prizes
	period   prize.Period
	text     string
	now      time.Time
	setErr   error
}

func (f *fakePrizeRepo) Get(_ context.Context, guildID string) (prize.Prizes, error) {
	return f.got, f.getErr
}

func (f *fakePrizeRepo) Upsert(_ context.Context, guildID string, period prize.Period, text string, now time.Time) (prize.Prizes, error) {
	f.period, f.text, f.now = period, text, now
	f.upserted = prize.Prizes{GuildID: guildID, Week: text, UpdatedAt: now}
	return f.upserted, f.setErr
}

var prizeAt = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

func newPrizeService(repo *fakePrizeRepo, guilds fakeGuilds) *service.PrizeService {
	return service.NewPrizeService(repo, guilds, fakeClock{now: prizeAt})
}

func TestPrizeServiceGet(t *testing.T) {
	repo := &fakePrizeRepo{got: prize.Prizes{GuildID: "g1", Week: "Cinema"}}

	p, err := newPrizeService(repo, fakeGuilds{guild: testGuild}).Get(context.Background(), "lia")

	require.NoError(t, err)
	assert.Equal(t, prize.Prizes{GuildID: "g1", Week: "Cinema"}, p)
}

func TestPrizeServiceGetRejectsNonMembers(t *testing.T) {
	repo := &fakePrizeRepo{}

	_, err := newPrizeService(repo, fakeGuilds{err: guild.ErrNotMember}).Get(context.Background(), "x")

	assert.ErrorIs(t, err, guild.ErrNotMember)
}

func TestPrizeServiceSet(t *testing.T) {
	repo := &fakePrizeRepo{}

	_, err := newPrizeService(repo, fakeGuilds{guild: testGuild}).Set(context.Background(), "lia", prize.PeriodWeek, "  Cinema  ")

	require.NoError(t, err)
	assert.Equal(t, prize.PeriodWeek, repo.period)
	assert.Equal(t, "Cinema", repo.text)
	assert.Equal(t, prizeAt, repo.now)
}

func TestPrizeServiceSetRejectsNonMembers(t *testing.T) {
	repo := &fakePrizeRepo{}

	_, err := newPrizeService(repo, fakeGuilds{err: guild.ErrNotMember}).Set(context.Background(), "x", prize.PeriodWeek, "Cinema")

	assert.ErrorIs(t, err, guild.ErrNotMember)
	assert.Empty(t, repo.text, "repository must not be touched")
}

func TestPrizeServiceSetRejectsTooLongBeforeRepo(t *testing.T) {
	repo := &fakePrizeRepo{}
	text := strings.Repeat("a", prize.MaxLen+1)

	_, err := newPrizeService(repo, fakeGuilds{guild: testGuild}).Set(context.Background(), "lia", prize.PeriodWeek, text)

	assert.ErrorIs(t, err, prize.ErrTooLong)
	assert.Empty(t, repo.text, "repository must not be touched")
}
