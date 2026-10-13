package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/push"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/reminder"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
)

// GuildLister lists every guild with its members.
type GuildLister interface {
	ListAll(ctx context.Context) ([]guild.Guild, error)
}

// ReminderRuleReader reads a guild's rules.
type ReminderRuleReader interface {
	ListByGuild(ctx context.Context, guildID string) ([]rule.Rule, error)
}

// ReminderEntryReader reads a member's entries.
type ReminderEntryReader interface {
	ListByMember(ctx context.Context, guildID, memberID string, from, to time.Time) ([]entry.Entry, error)
}

// ReminderLog records which users were reminded on which day.
type ReminderLog interface {
	// Claim inserts (userID, day) and reports true, or reports false if it already existed.
	Claim(ctx context.Context, userID string, day, now time.Time) (bool, error)
}

// PushSender delivers one payload to one subscription. It returns push.ErrGone
// when the push service answers 404 or 410.
type PushSender interface {
	Send(ctx context.Context, s push.Subscription, p reminder.Payload) error
}

// RunOptions tunes one daily run.
type RunOptions struct {
	// Force skips the 20h–23h send window.
	Force bool
	// UserID limits the run to that user and skips the once-a-day claim; "" means everyone.
	UserID string
}

// RunResult counts what one daily run did.
type RunResult struct {
	Day            time.Time // date-only
	UsersEligible  int       // has an unchecked daily quest and at least one device
	UsersNotified  int       // claimed (or forced) and had at least one device send attempted
	UsersSkipped   int       // eligible but already claimed today
	DevicesSent    int
	DevicesFailed  int
	DevicesRemoved int // push service said 404/410, row deleted
}

// ReminderService sends the daily "mark your quests" Web Push reminder.
type ReminderService struct {
	guilds  GuildLister
	rules   ReminderRuleReader
	entries ReminderEntryReader
	subs    PushSubscriptionRepository
	log     ReminderLog
	sender  PushSender
	clock   Clock
	loc     *time.Location
}

// NewReminderService builds a ReminderService. loc fixes the day boundary and the
// send window, the same location every other day-based use-case shares.
func NewReminderService(guilds GuildLister, rules ReminderRuleReader, entries ReminderEntryReader,
	subs PushSubscriptionRepository, log ReminderLog, sender PushSender, clock Clock, loc *time.Location) *ReminderService {
	return &ReminderService{guilds: guilds, rules: rules, entries: entries, subs: subs, log: log, sender: sender, clock: clock, loc: loc}
}

// RunDaily reminds every member who still has an unchecked daily quest today, once per
// day, on every device they subscribed. Repository errors abort the run; send errors don't.
func (s *ReminderService) RunDaily(ctx context.Context, opts RunOptions) (RunResult, error) {
	now := s.clock.Now()
	local := now.In(s.loc)
	if !opts.Force && !reminder.InWindow(local) {
		return RunResult{}, reminder.ErrOutsideWindow
	}
	res := RunResult{Day: entry.DateOnly(local)}
	guilds, err := s.guilds.ListAll(ctx)
	if err != nil {
		return RunResult{}, err
	}
	for _, g := range guilds {
		if err := s.remindGuild(ctx, g, opts, now, &res); err != nil {
			return RunResult{}, err
		}
	}
	return res, nil
}

func (s *ReminderService) remindGuild(ctx context.Context, g guild.Guild, opts RunOptions, now time.Time, res *RunResult) error {
	rules, err := s.rules.ListByGuild(ctx, g.ID)
	if err != nil {
		return err
	}
	var daily []string
	for _, r := range rules {
		if r.Frequency == rule.FrequencyDaily && r.ScoreType == rule.ScoreTypeSum {
			daily = append(daily, r.ID)
		}
	}
	if len(daily) == 0 {
		return nil
	}
	for _, member := range g.UserIDs {
		if opts.UserID != "" && member != opts.UserID {
			continue
		}
		if err := s.remindMember(ctx, g.ID, member, daily, opts, now, res); err != nil {
			return err
		}
	}
	return nil
}

func (s *ReminderService) remindMember(ctx context.Context, guildID, member string, daily []string, opts RunOptions, now time.Time, res *RunResult) error {
	entries, err := s.entries.ListByMember(ctx, guildID, member, res.Day, res.Day)
	if err != nil {
		return err
	}
	var done []string
	for _, e := range entries {
		if e.ScoreType == rule.ScoreTypeSum {
			done = append(done, e.RuleID)
		}
	}
	if !reminder.NeedsReminder(daily, done) {
		return nil
	}
	subs, err := s.subs.ListByUser(ctx, member)
	if err != nil {
		return err
	}
	if len(subs) == 0 {
		return nil
	}
	res.UsersEligible++
	if opts.UserID == "" {
		claimed, err := s.log.Claim(ctx, member, res.Day, now)
		if err != nil {
			return err
		}
		if !claimed {
			res.UsersSkipped++
			return nil
		}
	}
	res.UsersNotified++
	payload := reminder.Daily()
	for _, sub := range subs {
		err := s.sender.Send(ctx, sub, payload)
		switch {
		case err == nil:
			res.DevicesSent++
		case errors.Is(err, push.ErrGone):
			// Only this member's row: the endpoint may have just moved to another user.
			if err := s.subs.DeleteForUser(ctx, member, sub.Endpoint); err != nil {
				return err
			}
			res.DevicesRemoved++
		default:
			res.DevicesFailed++
			log.Printf("daily reminder: send to user=%s host=%s failed: %v", member, sub.Host(), err)
		}
	}
	return nil
}
