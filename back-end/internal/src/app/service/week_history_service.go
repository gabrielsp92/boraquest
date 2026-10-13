package service

import (
	"context"
	"time"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/weekresult"
)

// WeekResultRepository persists closed weeks, always scoped to the guild.
type WeekResultRepository interface {
	// LatestWeekStart returns the week_start of the guild's most recent closed
	// week, or false if none has been closed yet.
	LatestWeekStart(ctx context.Context, guildID string) (time.Time, bool, error)
	// SaveAll stores results atomically; weeks already stored are left untouched,
	// so concurrent closes of the same week are harmless.
	SaveAll(ctx context.Context, results []weekresult.WeekResult) error
	// List returns every closed week of the guild, newest first, standings by rank.
	List(ctx context.Context, guildID string) ([]weekresult.WeekResult, error)
}

// WeekEntryReader is the slice of entry storage that closing weeks needs.
type WeekEntryReader interface {
	SumByGuild(ctx context.Context, guildID string, from, to time.Time) ([]MemberTotal, error)
	// WeekStartsWithEntries returns, ascending, the Monday of every week that has
	// at least one of the guild's entries with occurredOn in [from, before).
	WeekStartsWithEntries(ctx context.Context, guildID string, from, before time.Time) ([]time.Time, error)
}

// PrizeHistoryReader reads the log of past prize edits.
type PrizeHistoryReader interface {
	// WeekPrizeAt returns the guild's week prize as it was right before at,
	// or "" if none had been set by then.
	WeekPrizeAt(ctx context.Context, guildID string, at time.Time) (string, error)
}

// closeGrace delays closing a week past its end, so an entry validated for
// Sunday just before midnight still commits before the week is frozen.
const closeGrace = 5 * time.Minute

// WeekHistoryService closes ended weeks (lazily, on read) and lists them.
// Entries can only be logged for today, so a week's totals are final once it ends.
type WeekHistoryService struct {
	results WeekResultRepository
	entries WeekEntryReader
	prizes  PrizeHistoryReader
	guilds  GuildRepository
	clock   Clock
	loc     *time.Location
}

// NewWeekHistoryService builds a WeekHistoryService. loc fixes the week boundary.
func NewWeekHistoryService(results WeekResultRepository, entries WeekEntryReader, prizes PrizeHistoryReader, guilds GuildRepository, clock Clock, loc *time.Location) *WeekHistoryService {
	return &WeekHistoryService{results: results, entries: entries, prizes: prizes, guilds: guilds, clock: clock, loc: loc}
}

// List closes every ended week of callerID's guild not yet recorded, then
// returns all closed weeks, newest first.
func (s *WeekHistoryService) List(ctx context.Context, callerID string) ([]weekresult.WeekResult, error) {
	g, err := s.guilds.FindByMember(ctx, callerID)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	current := entry.WeekStart(entry.DateOnly(now.Add(-closeGrace).In(s.loc)))
	var from time.Time // zero: no week closed yet, scan from the very first entry
	if latest, ok, err := s.results.LatestWeekStart(ctx, g.ID); err != nil {
		return nil, err
	} else if ok {
		from = latest.AddDate(0, 0, 7)
	}
	weeks, err := s.entries.WeekStartsWithEntries(ctx, g.ID, from, current)
	if err != nil {
		return nil, err
	}
	if len(weeks) > 0 {
		closed := make([]weekresult.WeekResult, 0, len(weeks))
		for _, ws := range weeks {
			r, err := s.close(ctx, g.ID, g.UserIDs, ws, now)
			if err != nil {
				return nil, err
			}
			closed = append(closed, r)
		}
		if err := s.results.SaveAll(ctx, closed); err != nil {
			return nil, err
		}
	}
	return s.results.List(ctx, g.ID)
}

// close freezes the week starting at ws: ranked standings (current members
// zero-filled, like the scoreboard) and the week prize in effect when it ended.
func (s *WeekHistoryService) close(ctx context.Context, guildID string, memberIDs []string, ws, now time.Time) (weekresult.WeekResult, error) {
	we := ws.AddDate(0, 0, 6)
	totals, err := s.entries.SumByGuild(ctx, guildID, ws, we)
	if err != nil {
		return weekresult.WeekResult{}, err
	}
	byMember := make(map[string]MemberTotal, len(totals))
	for _, t := range totals {
		byMember[t.MemberID] = t
	}
	standings := make([]Standing, 0, len(memberIDs)+len(totals))
	for _, id := range memberIDs {
		t := byMember[id]
		delete(byMember, id)
		standings = append(standings, Standing{MemberID: id, Points: t.Points, Completed: t.Completed})
	}
	for _, t := range byMember { // scored that week but no longer in the guild
		standings = append(standings, Standing{MemberID: t.MemberID, Points: t.Points, Completed: t.Completed})
	}
	rankStandings(standings)

	next := ws.AddDate(0, 0, 7)
	end := time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, s.loc) // Monday 00:00 local
	prize, err := s.prizes.WeekPrizeAt(ctx, guildID, end)
	if err != nil {
		return weekresult.WeekResult{}, err
	}

	r := weekresult.WeekResult{GuildID: guildID, WeekStart: ws, WeekEnd: we, Prize: prize, ClosedAt: now, Standings: make([]weekresult.Standing, 0, len(standings))}
	for i, st := range standings {
		r.Standings = append(r.Standings, weekresult.Standing{MemberID: st.MemberID, Rank: i + 1, Points: st.Points, Completed: st.Completed})
	}
	return r, nil
}
