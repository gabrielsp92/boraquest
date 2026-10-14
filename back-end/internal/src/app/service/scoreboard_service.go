package service

import (
	"context"
	"slices"
	"sort"
	"time"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
)

type ScoreboardPeriod string

const (
	ScoreboardWeek  ScoreboardPeriod = "week"
	ScoreboardMonth ScoreboardPeriod = "month"
)

// MemberTotal is one guild member's raw aggregate over a period, as read from storage.
// Members with zero entries in the period are absent from the repository's result;
// ScoreboardService fills them in at zero (Business rule 4).
type MemberTotal struct {
	MemberID  string
	Points    int
	Completed int
}

// Standing is one ranked row of the scoreboard.
type Standing struct {
	MemberID  string
	Points    int
	Completed int
}

// QuestTotal is one member's aggregate for one quest over a period.
type QuestTotal struct {
	RuleID    string
	RuleName  string
	ScoreType rule.ScoreType
	Count     int
	Points    int
}

// ScoreboardService aggregates entries into one ranked row per guild member.
type ScoreboardService struct {
	entries EntryRepository
	guilds  GuildRepository
	clock   Clock
	loc     *time.Location
}

// NewScoreboardService builds a ScoreboardService.
func NewScoreboardService(entries EntryRepository, guilds GuildRepository, clock Clock, loc *time.Location) *ScoreboardService {
	return &ScoreboardService{entries: entries, guilds: guilds, clock: clock, loc: loc}
}

// List returns callerID's guild's standings for period, along with the date range
// covered and today's date, all computed in the guild's fixed timezone.
func (s *ScoreboardService) List(ctx context.Context, callerID string, period ScoreboardPeriod) (standings []Standing, from, to, today time.Time, err error) {
	g, err := s.guilds.FindByMember(ctx, callerID)
	if err != nil {
		return nil, time.Time{}, time.Time{}, time.Time{}, err
	}
	today = entry.DateOnly(s.clock.Now().In(s.loc))
	from, to = periodRange(today, period)
	totals, err := s.entries.SumByGuild(ctx, g.ID, from, to)
	if err != nil {
		return nil, time.Time{}, time.Time{}, time.Time{}, err
	}
	byMember := make(map[string]MemberTotal, len(totals))
	for _, t := range totals {
		byMember[t.MemberID] = t
	}
	standings = make([]Standing, 0, len(g.UserIDs))
	for _, id := range g.UserIDs {
		t := byMember[id] // zero value when the member has no entries this period
		standings = append(standings, Standing{MemberID: id, Points: t.Points, Completed: t.Completed})
	}
	rankStandings(standings)
	return standings, from, to, today, nil
}

// Breakdown returns memberID's per-quest totals for period, sorted by points
// descending then name ascending, with the range covered and today. The range is
// the same as List's, so the totals always match the member's standing.
// Errors: guild.ErrNotMember (caller has no guild), entry.ErrMemberNotInGuild
// (memberID not in the caller's guild).
func (s *ScoreboardService) Breakdown(ctx context.Context, callerID, memberID string, period ScoreboardPeriod) (items []QuestTotal, from, to, today time.Time, err error) {
	g, err := s.guilds.FindByMember(ctx, callerID)
	if err != nil {
		return nil, time.Time{}, time.Time{}, time.Time{}, err
	}
	if !slices.Contains(g.UserIDs, memberID) {
		return nil, time.Time{}, time.Time{}, time.Time{}, entry.ErrMemberNotInGuild
	}
	today = entry.DateOnly(s.clock.Now().In(s.loc))
	from, to = periodRange(today, period)
	items, err = s.entries.SumByRule(ctx, g.ID, memberID, from, to)
	if err != nil {
		return nil, time.Time{}, time.Time{}, time.Time{}, err
	}
	names := collate.New(language.BrazilianPortuguese) // "Água" sorts with the A's
	sort.Slice(items, func(i, j int) bool {
		if items[i].Points != items[j].Points {
			return items[i].Points > items[j].Points
		}
		if c := names.CompareString(items[i].RuleName, items[j].RuleName); c != 0 {
			return c < 0
		}
		// Same points and name (e.g. a quest re-created after deletion): keep a stable order.
		if items[i].RuleID != items[j].RuleID {
			return items[i].RuleID < items[j].RuleID
		}
		return items[i].ScoreType < items[j].ScoreType
	})
	return items, from, to, today, nil
}

// periodRange returns the first and last day of period around today: Monday to
// Sunday for a week, the 1st to the last day for a month.
func periodRange(today time.Time, period ScoreboardPeriod) (from, to time.Time) {
	if period == ScoreboardMonth {
		from = time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
		return from, from.AddDate(0, 1, 0).AddDate(0, 0, -1) // last day of the month
	}
	from = entry.WeekStart(today)
	return from, from.AddDate(0, 0, 6)
}

// rankStandings sorts standings by points descending, then tasks completed
// descending, then member id ascending — a fully deterministic order shared by
// the live scoreboard and the closed-week history, so both always agree.
func rankStandings(standings []Standing) {
	sort.Slice(standings, func(i, j int) bool {
		if standings[i].Points != standings[j].Points {
			return standings[i].Points > standings[j].Points
		}
		if standings[i].Completed != standings[j].Completed {
			return standings[i].Completed > standings[j].Completed
		}
		return standings[i].MemberID < standings[j].MemberID
	})
}
