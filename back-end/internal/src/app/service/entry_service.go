package service

import (
	"context"
	"time"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
)

// EntryInput holds the fields needed to create an entry.
type EntryInput struct {
	RuleID   string
	MemberID string
}

// EntryPeriod selects the date range an entry list query covers.
type EntryPeriod string

const (
	PeriodToday EntryPeriod = "today"
	PeriodWeek  EntryPeriod = "week"
)

// EntryRepository persists entries, always scoped to a guild.
type EntryRepository interface {
	// Create inserts e. For e.ScoreType == rule.ScoreTypeSum it enforces one entry per
	// (rule, member, period_key); returns entry.ErrAlreadyLogged when one already exists.
	Create(ctx context.Context, e entry.Entry) error
	// Get returns entry.ErrNotFound when no entry with id exists in the guild.
	Get(ctx context.Context, guildID, id string) (entry.Entry, error)
	// Delete returns entry.ErrNotFound when no entry with id exists in the guild.
	Delete(ctx context.Context, guildID, id string) error
	// ListByMember returns a member's entries with occurredOn in [from, to], newest first.
	ListByMember(ctx context.Context, guildID, memberID string, from, to time.Time) ([]entry.Entry, error)
}

// EntryService manages entries logged by members of the caller's guild.
type EntryService struct {
	entries EntryRepository
	rules   RuleRepository // reused interface from rule_service.go
	guilds  GuildRepository
	ids     IDGenerator
	clock   Clock
	loc     *time.Location
}

// NewEntryService builds an EntryService. loc fixes the day/week boundary every
// caller shares, regardless of each device's own timezone (see package docs).
func NewEntryService(entries EntryRepository, rules RuleRepository, guilds GuildRepository, ids IDGenerator, clock Clock, loc *time.Location) *EntryService {
	return &EntryService{entries: entries, rules: rules, guilds: guilds, ids: ids, clock: clock, loc: loc}
}

// Create logs an entry on behalf of callerID, scoped to its guild.
func (s *EntryService) Create(ctx context.Context, callerID string, in EntryInput) (entry.Entry, error) {
	g, err := s.guilds.FindByMember(ctx, callerID)
	if err != nil {
		return entry.Entry{}, err
	}
	r, err := s.rules.Get(ctx, g.ID, in.RuleID)
	if err != nil {
		return entry.Entry{}, err // rule.ErrNotFound
	}
	if in.MemberID == "" || !g.HasMember(in.MemberID) {
		return entry.Entry{}, entry.ErrMemberNotInGuild
	}
	now := s.clock.Now().In(s.loc)
	e, err := entry.New(s.ids.NewID(), g.ID, r.ID, r.Name, r.ScoreType, r.Score, in.MemberID, callerID, r.Frequency, now)
	if err != nil {
		return entry.Entry{}, err // entry.ErrWrongMember
	}
	if err := s.entries.Create(ctx, e); err != nil {
		return entry.Entry{}, err // entry.ErrAlreadyLogged
	}
	return e, nil
}

// Delete removes a sum-type entry callerID created today.
func (s *EntryService) Delete(ctx context.Context, callerID, id string) error {
	g, err := s.guilds.FindByMember(ctx, callerID)
	if err != nil {
		return err
	}
	e, err := s.entries.Get(ctx, g.ID, id)
	if err != nil {
		return err // entry.ErrNotFound
	}
	if e.MemberID != callerID {
		return entry.ErrNotOwn
	}
	if e.ScoreType != rule.ScoreTypeSum {
		return entry.ErrNotRemovable
	}
	if !e.OccurredOn.Equal(entry.DateOnly(s.clock.Now().In(s.loc))) {
		return entry.ErrNotToday
	}
	return s.entries.Delete(ctx, g.ID, id)
}

// List returns callerID's entries for period, along with the date range covered
// and today's date, all computed in the guild's fixed timezone.
func (s *EntryService) List(ctx context.Context, callerID string, period EntryPeriod) (entries []entry.Entry, from, to, today time.Time, err error) {
	g, err := s.guilds.FindByMember(ctx, callerID)
	if err != nil {
		return nil, time.Time{}, time.Time{}, time.Time{}, err
	}
	today = entry.DateOnly(s.clock.Now().In(s.loc))
	if period == PeriodWeek {
		from = entry.WeekStart(today)
		to = from.AddDate(0, 0, 6)
	} else {
		from, to = today, today
	}
	entries, err = s.entries.ListByMember(ctx, g.ID, callerID, from, to)
	return entries, from, to, today, err
}
