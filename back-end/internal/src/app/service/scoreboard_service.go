package service

import (
	"context"
	"sort"
	"time"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
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
	if period == ScoreboardMonth {
		from = time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
		to = from.AddDate(0, 1, 0).AddDate(0, 0, -1) // last day of the month
	} else {
		from = entry.WeekStart(today)
		to = from.AddDate(0, 0, 6)
	}
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
	sort.Slice(standings, func(i, j int) bool {
		if standings[i].Points != standings[j].Points {
			return standings[i].Points > standings[j].Points
		}
		if standings[i].Completed != standings[j].Completed {
			return standings[i].Completed > standings[j].Completed
		}
		return standings[i].MemberID < standings[j].MemberID
	})
	return standings, from, to, today, nil
}
