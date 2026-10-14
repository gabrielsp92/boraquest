package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
)

var scoreboardGuild = guild.Guild{ID: "g1", Name: "Família", UserIDs: []string{"lia", "beto", "nena", "caio"}}

func newScoreboardService(entries *fakeEntryRepo, guilds fakeGuilds, now time.Time) *service.ScoreboardService {
	return service.NewScoreboardService(entries, guilds, fakeClock{now: now}, saoPaulo)
}

func TestScoreboardServiceZeroFillsMembersWithNoEntries(t *testing.T) {
	entries := &fakeEntryRepo{summed: []service.MemberTotal{{MemberID: "lia", Points: 30, Completed: 3}}}
	at := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC) // Thursday

	standings, _, _, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, at).List(context.Background(), "lia", service.ScoreboardWeek)

	require.NoError(t, err)
	require.Len(t, standings, 4)
	byID := make(map[string]service.Standing, len(standings))
	for _, s := range standings {
		byID[s.MemberID] = s
	}
	assert.Equal(t, service.Standing{MemberID: "lia", Points: 30, Completed: 3}, byID["lia"])
	assert.Equal(t, service.Standing{MemberID: "beto", Points: 0, Completed: 0}, byID["beto"])
	assert.Equal(t, service.Standing{MemberID: "nena", Points: 0, Completed: 0}, byID["nena"])
	assert.Equal(t, service.Standing{MemberID: "caio", Points: 0, Completed: 0}, byID["caio"])
}

func TestScoreboardServiceSortsByPointsDescending(t *testing.T) {
	entries := &fakeEntryRepo{summed: []service.MemberTotal{
		{MemberID: "lia", Points: 30, Completed: 3},
		{MemberID: "beto", Points: 50, Completed: 2},
	}}

	standings, _, _, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, time.Now()).List(context.Background(), "lia", service.ScoreboardWeek)

	require.NoError(t, err)
	assert.Equal(t, "beto", standings[0].MemberID)
	assert.Equal(t, "lia", standings[1].MemberID)
}

func TestScoreboardServiceTieBreaksByCompletedThenMemberID(t *testing.T) {
	entries := &fakeEntryRepo{summed: []service.MemberTotal{
		{MemberID: "nena", Points: 20, Completed: 1},
		{MemberID: "caio", Points: 20, Completed: 2},
		{MemberID: "beto", Points: 20, Completed: 2},
	}}

	standings, _, _, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, time.Now()).List(context.Background(), "lia", service.ScoreboardWeek)

	require.NoError(t, err)
	ids := []string{standings[0].MemberID, standings[1].MemberID, standings[2].MemberID, standings[3].MemberID}
	// beto, caio tied at 20/2 -> member id ascending; nena at 20/1; lia at 0/0 last.
	assert.Equal(t, []string{"beto", "caio", "nena", "lia"}, ids)
}

func TestScoreboardServiceNegativePointsAllowed(t *testing.T) {
	entries := &fakeEntryRepo{summed: []service.MemberTotal{{MemberID: "beto", Points: -15, Completed: 0}}}

	standings, _, _, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, time.Now()).List(context.Background(), "lia", service.ScoreboardWeek)

	require.NoError(t, err)
	byID := make(map[string]service.Standing, len(standings))
	for _, s := range standings {
		byID[s.MemberID] = s
	}
	assert.Equal(t, -15, byID["beto"].Points)
}

func TestScoreboardServiceWeekRange(t *testing.T) {
	entries := &fakeEntryRepo{}
	at := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC) // Thursday in Sao Paulo

	_, from, to, today, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, at).List(context.Background(), "lia", service.ScoreboardWeek)

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), from)
	assert.Equal(t, time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC), to)
	assert.Equal(t, time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC), today)
	assert.Equal(t, "g1", entries.sumArgs.guildID)
	assert.Equal(t, from, entries.sumArgs.from)
	assert.Equal(t, to, entries.sumArgs.to)
}

func TestScoreboardServiceMonthRange31Days(t *testing.T) {
	entries := &fakeEntryRepo{}
	at := time.Date(2026, 10, 15, 15, 0, 0, 0, time.UTC)

	_, from, to, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, at).List(context.Background(), "lia", service.ScoreboardMonth)

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), from)
	assert.Equal(t, time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), to)
}

func TestScoreboardServiceMonthRange28DaysFebruary(t *testing.T) {
	entries := &fakeEntryRepo{}
	at := time.Date(2027, 2, 10, 15, 0, 0, 0, time.UTC) // 2027 is not a leap year

	_, from, to, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, at).List(context.Background(), "lia", service.ScoreboardMonth)

	require.NoError(t, err)
	assert.Equal(t, time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC), from)
	assert.Equal(t, time.Date(2027, 2, 28, 0, 0, 0, 0, time.UTC), to)
}

func TestScoreboardServiceMonthRange29DaysLeapYear(t *testing.T) {
	entries := &fakeEntryRepo{}
	at := time.Date(2028, 2, 10, 15, 0, 0, 0, time.UTC) // 2028 is a leap year

	_, from, to, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, at).List(context.Background(), "lia", service.ScoreboardMonth)

	require.NoError(t, err)
	assert.Equal(t, time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC), from)
	assert.Equal(t, time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC), to)
}

func TestScoreboardServiceMonthRange30DaysApril(t *testing.T) {
	entries := &fakeEntryRepo{}
	at := time.Date(2026, 4, 15, 15, 0, 0, 0, time.UTC)

	_, from, to, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, at).List(context.Background(), "lia", service.ScoreboardMonth)

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), from)
	assert.Equal(t, time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC), to)
}

func TestScoreboardServiceDecemberToJanuaryBoundary(t *testing.T) {
	entries := &fakeEntryRepo{}
	at := time.Date(2026, 12, 20, 15, 0, 0, 0, time.UTC)

	_, from, to, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, at).List(context.Background(), "lia", service.ScoreboardMonth)

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC), from)
	assert.Equal(t, time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), to)
}

func TestScoreboardServiceRejectsNonMember(t *testing.T) {
	entries := &fakeEntryRepo{}

	_, _, _, _, err := newScoreboardService(entries, fakeGuilds{err: guild.ErrNotMember}, time.Now()).List(context.Background(), "x", service.ScoreboardWeek)

	assert.ErrorIs(t, err, guild.ErrNotMember)
}

func TestScoreboardServiceRepoError(t *testing.T) {
	entries := &fakeEntryRepo{sumErr: errBoom}

	_, _, _, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, time.Now()).List(context.Background(), "lia", service.ScoreboardWeek)

	assert.ErrorIs(t, err, errBoom)
}

func TestScoreboardServiceBreakdownSortsByPointsThenName(t *testing.T) {
	entries := &fakeEntryRepo{byRule: []service.QuestTotal{
		{RuleID: "r9", RuleName: "Esquecer a louça", ScoreType: rule.ScoreTypeDecrease, Count: 1, Points: -10},
		{RuleID: "r3", RuleName: "Treino", ScoreType: rule.ScoreTypeSum, Count: 2, Points: 30},
		{RuleID: "r1", RuleName: "Dormir cedo", ScoreType: rule.ScoreTypeSum, Count: 5, Points: 50},
		{RuleID: "r2", RuleName: "Ler", ScoreType: rule.ScoreTypeSum, Count: 3, Points: 30},
		{RuleID: "r4", RuleName: "Água", ScoreType: rule.ScoreTypeSum, Count: 3, Points: 30},
	}}
	at := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC) // Thursday

	items, _, _, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, at).Breakdown(context.Background(), "lia", "beto", service.ScoreboardWeek)

	require.NoError(t, err)
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.RuleName
	}
	// Ties by name use Portuguese collation, so "Água" comes before "Ler", not after "Treino".
	assert.Equal(t, []string{"Dormir cedo", "Água", "Ler", "Treino", "Esquecer a louça"}, names)
	assert.Equal(t, "g1", entries.byRuleArgs.guildID)
	assert.Equal(t, "beto", entries.byRuleArgs.memberID)
}

func TestScoreboardServiceBreakdownOwnCard(t *testing.T) {
	entries := &fakeEntryRepo{}

	items, _, _, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, time.Now()).Breakdown(context.Background(), "lia", "lia", service.ScoreboardWeek)

	require.NoError(t, err)
	assert.Empty(t, items)
	assert.Equal(t, "lia", entries.byRuleArgs.memberID)
}

func TestScoreboardServiceBreakdownWeekRangeOnSunday(t *testing.T) {
	entries := &fakeEntryRepo{}
	at := time.Date(2026, 10, 12, 2, 0, 0, 0, time.UTC) // Sunday 23:00 in Sao Paulo

	_, from, to, today, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, at).Breakdown(context.Background(), "lia", "beto", service.ScoreboardWeek)

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), from)
	assert.Equal(t, time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC), to)
	assert.Equal(t, time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC), today)
	assert.Equal(t, from, entries.byRuleArgs.from)
	assert.Equal(t, to, entries.byRuleArgs.to)
}

func TestScoreboardServiceBreakdownMonthRange(t *testing.T) {
	for name, at := range map[string]time.Time{
		"first day": time.Date(2026, 10, 1, 15, 0, 0, 0, time.UTC),
		"last day":  time.Date(2026, 11, 1, 2, 0, 0, 0, time.UTC), // Oct 31 23:00 in Sao Paulo
	} {
		t.Run(name, func(t *testing.T) {
			entries := &fakeEntryRepo{}

			_, from, to, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, at).Breakdown(context.Background(), "lia", "beto", service.ScoreboardMonth)

			require.NoError(t, err)
			assert.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), from)
			assert.Equal(t, time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), to)
			assert.Equal(t, from, entries.byRuleArgs.from)
			assert.Equal(t, to, entries.byRuleArgs.to)
		})
	}
}

func TestScoreboardServiceBreakdownRejectsCallerWithoutGuild(t *testing.T) {
	entries := &fakeEntryRepo{}

	_, _, _, _, err := newScoreboardService(entries, fakeGuilds{err: guild.ErrNotMember}, time.Now()).Breakdown(context.Background(), "x", "lia", service.ScoreboardWeek)

	assert.ErrorIs(t, err, guild.ErrNotMember)
}

func TestScoreboardServiceBreakdownRejectsMemberOutsideGuild(t *testing.T) {
	entries := &fakeEntryRepo{}

	_, _, _, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, time.Now()).Breakdown(context.Background(), "lia", "intruso", service.ScoreboardWeek)

	assert.ErrorIs(t, err, entry.ErrMemberNotInGuild)
	assert.Empty(t, entries.byRuleArgs.memberID, "repository must not be queried")
}

func TestScoreboardServiceBreakdownRepoError(t *testing.T) {
	entries := &fakeEntryRepo{byRuleErr: errBoom}

	_, _, _, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, time.Now()).Breakdown(context.Background(), "lia", "beto", service.ScoreboardWeek)

	assert.ErrorIs(t, err, errBoom)
}

func TestScoreboardServiceBreakdownTieBreaksByRuleIDThenScoreType(t *testing.T) {
	entries := &fakeEntryRepo{byRule: []service.QuestTotal{
		{RuleID: "r2", RuleName: "Treino", ScoreType: rule.ScoreTypeSum, Count: 1, Points: 0},
		{RuleID: "r1", RuleName: "Treino", ScoreType: rule.ScoreTypeSum, Count: 1, Points: 0},
		{RuleID: "r1", RuleName: "Treino", ScoreType: rule.ScoreTypeDecrease, Count: 1, Points: 0},
	}}

	items, _, _, _, err := newScoreboardService(entries, fakeGuilds{guild: scoreboardGuild}, time.Now()).Breakdown(context.Background(), "lia", "beto", service.ScoreboardWeek)

	require.NoError(t, err)
	assert.Equal(t, []service.QuestTotal{
		{RuleID: "r1", RuleName: "Treino", ScoreType: rule.ScoreTypeDecrease, Count: 1, Points: 0},
		{RuleID: "r1", RuleName: "Treino", ScoreType: rule.ScoreTypeSum, Count: 1, Points: 0},
		{RuleID: "r2", RuleName: "Treino", ScoreType: rule.ScoreTypeSum, Count: 1, Points: 0},
	}, items)
}
