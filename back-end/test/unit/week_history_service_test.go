package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/weekresult"
)

type fakeWeekResults struct {
	latest    time.Time
	hasLatest bool
	latestErr error
	saveErr   error
	listed    []weekresult.WeekResult
	listErr   error
	saved     []weekresult.WeekResult
	saveCalls int
}

func (f *fakeWeekResults) LatestWeekStart(context.Context, string) (time.Time, bool, error) {
	return f.latest, f.hasLatest, f.latestErr
}

func (f *fakeWeekResults) SaveAll(_ context.Context, results []weekresult.WeekResult) error {
	f.saveCalls++
	f.saved = results
	return f.saveErr
}

func (f *fakeWeekResults) List(context.Context, string) ([]weekresult.WeekResult, error) {
	return f.listed, f.listErr
}

type fakeWeekEntries struct {
	weeks        []time.Time
	weeksErr     error
	from, before time.Time
	sums         map[string][]service.MemberTotal // keyed by week start "2006-01-02"
	sumErr       error
	sumRanges    [][2]time.Time
}

func (f *fakeWeekEntries) WeekStartsWithEntries(_ context.Context, _ string, from, before time.Time) ([]time.Time, error) {
	f.from, f.before = from, before
	return f.weeks, f.weeksErr
}

func (f *fakeWeekEntries) SumByGuild(_ context.Context, _ string, from, to time.Time) ([]service.MemberTotal, error) {
	f.sumRanges = append(f.sumRanges, [2]time.Time{from, to})
	return f.sums[from.Format("2006-01-02")], f.sumErr
}

type fakePrizeHistory struct {
	text string
	err  error
	ats  []time.Time
}

func (f *fakePrizeHistory) WeekPrizeAt(_ context.Context, _ string, at time.Time) (string, error) {
	f.ats = append(f.ats, at)
	return f.text, f.err
}

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// Thursday 2026-10-08, 12:00 in São Paulo: the current week starts Monday 2026-10-05.
var historyNow = time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)

func newWeekHistoryService(results *fakeWeekResults, entries *fakeWeekEntries, prizes *fakePrizeHistory, guilds fakeGuilds, now time.Time) *service.WeekHistoryService {
	return service.NewWeekHistoryService(results, entries, prizes, guilds, fakeClock{now: now}, saoPaulo)
}

func TestWeekHistoryServiceBackfillsEveryEndedWeekWithEntries(t *testing.T) {
	results := &fakeWeekResults{listed: []weekresult.WeekResult{{GuildID: "g1"}}}
	entries := &fakeWeekEntries{
		weeks: []time.Time{day(2026, 9, 14), day(2026, 9, 28)}, // the week of 09-21 had no entries
		sums: map[string][]service.MemberTotal{
			"2026-09-14": {{MemberID: "beto", Points: 25, Completed: 3}},
			"2026-09-28": {{MemberID: "nena", Points: 40, Completed: 4}, {MemberID: "lia", Points: 10, Completed: 1}},
		},
	}
	prizes := &fakePrizeHistory{text: "Escolher o filme"}

	got, err := newWeekHistoryService(results, entries, prizes, fakeGuilds{guild: scoreboardGuild}, historyNow).List(context.Background(), "lia")

	require.NoError(t, err)
	assert.Equal(t, results.listed, got)
	assert.True(t, entries.from.IsZero(), "no closed week yet: scan from the first entry")
	assert.Equal(t, day(2026, 10, 5), entries.before, "the current week is never closed")
	assert.Equal(t, [][2]time.Time{{day(2026, 9, 14), day(2026, 9, 20)}, {day(2026, 9, 28), day(2026, 10, 4)}}, entries.sumRanges)
	// The prize is read as of Monday 00:00 in São Paulo (03:00 UTC) after each week.
	assert.Equal(t, []time.Time{
		time.Date(2026, 9, 21, 3, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 5, 3, 0, 0, 0, time.UTC),
	}, []time.Time{prizes.ats[0].UTC(), prizes.ats[1].UTC()})

	require.Equal(t, 1, results.saveCalls)
	require.Len(t, results.saved, 2)
	first, second := results.saved[0], results.saved[1]
	assert.Equal(t, "g1", first.GuildID)
	assert.Equal(t, day(2026, 9, 14), first.WeekStart)
	assert.Equal(t, day(2026, 9, 20), first.WeekEnd)
	assert.Equal(t, "Escolher o filme", first.Prize)
	assert.Equal(t, historyNow, first.ClosedAt)
	assert.Equal(t, weekresult.Standing{MemberID: "beto", Rank: 1, Points: 25, Completed: 3}, first.Standings[0])
	assert.Equal(t, []weekresult.Standing{
		{MemberID: "nena", Rank: 1, Points: 40, Completed: 4},
		{MemberID: "lia", Rank: 2, Points: 10, Completed: 1},
		{MemberID: "beto", Rank: 3},
		{MemberID: "caio", Rank: 4},
	}, second.Standings)
}

func TestWeekHistoryServiceResumesAfterLatestClosedWeek(t *testing.T) {
	results := &fakeWeekResults{latest: day(2026, 9, 21), hasLatest: true}
	entries := &fakeWeekEntries{}

	_, err := newWeekHistoryService(results, entries, &fakePrizeHistory{}, fakeGuilds{guild: scoreboardGuild}, historyNow).List(context.Background(), "lia")

	require.NoError(t, err)
	assert.Equal(t, day(2026, 9, 28), entries.from)
	assert.Equal(t, 0, results.saveCalls, "nothing new to close")
}

func TestWeekHistoryServiceUsesGuildTimezoneForCurrentWeek(t *testing.T) {
	entries := &fakeWeekEntries{}
	// Monday 02:00 UTC is still Sunday 23:00 in São Paulo: that week hasn't ended.
	sundayNight := time.Date(2026, 10, 12, 2, 0, 0, 0, time.UTC)

	_, err := newWeekHistoryService(&fakeWeekResults{}, entries, &fakePrizeHistory{}, fakeGuilds{guild: scoreboardGuild}, sundayNight).List(context.Background(), "lia")

	require.NoError(t, err)
	assert.Equal(t, day(2026, 10, 5), entries.before)
}

func TestWeekHistoryServiceRanksWithScoreboardTieBreakAndKeepsFormerMembers(t *testing.T) {
	results := &fakeWeekResults{}
	entries := &fakeWeekEntries{
		weeks: []time.Time{day(2026, 9, 28)},
		sums: map[string][]service.MemberTotal{"2026-09-28": {
			{MemberID: "nena", Points: 20, Completed: 1},
			{MemberID: "caio", Points: 20, Completed: 2},
			{MemberID: "beto", Points: 20, Completed: 2},
			{MemberID: "ex", Points: -5, Completed: 0}, // scored that week, has since left the guild
		}},
	}

	_, err := newWeekHistoryService(results, entries, &fakePrizeHistory{}, fakeGuilds{guild: scoreboardGuild}, historyNow).List(context.Background(), "lia")

	require.NoError(t, err)
	var ids []string
	for _, s := range results.saved[0].Standings {
		ids = append(ids, s.MemberID)
	}
	assert.Equal(t, []string{"beto", "caio", "nena", "lia", "ex"}, ids)
	assert.Equal(t, "", results.saved[0].Prize, "no prize set by then")
}

func TestWeekHistoryServiceErrors(t *testing.T) {
	oneWeek := []time.Time{day(2026, 9, 28)}
	cases := map[string]struct {
		guilds  fakeGuilds
		results *fakeWeekResults
		entries *fakeWeekEntries
		prizes  *fakePrizeHistory
		want    error
	}{
		"not a member": {fakeGuilds{err: guild.ErrNotMember}, &fakeWeekResults{}, &fakeWeekEntries{}, &fakePrizeHistory{}, guild.ErrNotMember},
		"latest":       {fakeGuilds{guild: scoreboardGuild}, &fakeWeekResults{latestErr: errBoom}, &fakeWeekEntries{}, &fakePrizeHistory{}, errBoom},
		"weeks":        {fakeGuilds{guild: scoreboardGuild}, &fakeWeekResults{}, &fakeWeekEntries{weeksErr: errBoom}, &fakePrizeHistory{}, errBoom},
		"sum":          {fakeGuilds{guild: scoreboardGuild}, &fakeWeekResults{}, &fakeWeekEntries{weeks: oneWeek, sumErr: errBoom}, &fakePrizeHistory{}, errBoom},
		"prize":        {fakeGuilds{guild: scoreboardGuild}, &fakeWeekResults{}, &fakeWeekEntries{weeks: oneWeek}, &fakePrizeHistory{err: errBoom}, errBoom},
		"save":         {fakeGuilds{guild: scoreboardGuild}, &fakeWeekResults{saveErr: errBoom}, &fakeWeekEntries{weeks: oneWeek}, &fakePrizeHistory{}, errBoom},
		"list":         {fakeGuilds{guild: scoreboardGuild}, &fakeWeekResults{listErr: errBoom}, &fakeWeekEntries{}, &fakePrizeHistory{}, errBoom},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := newWeekHistoryService(tc.results, tc.entries, tc.prizes, tc.guilds, historyNow).List(context.Background(), "lia")

			assert.ErrorIs(t, err, tc.want)
			assert.Nil(t, got)
		})
	}
}
