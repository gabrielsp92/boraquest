// Package entry holds the domain model for entries: a rule scored for a
// guild member on a given day (a checked-off quest, or a logged slip).
package entry

import (
	"errors"
	"slices"
	"time"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
)

var (
	ErrNotFound         = errors.New("entry: not found")
	ErrAlreadyLogged    = errors.New("entry: already logged for this period")
	ErrWrongMember      = errors.New("entry: you can only complete a quest for yourself")
	ErrMemberNotInGuild = errors.New("entry: member is not in your guild")
	ErrNotOwn           = errors.New("entry: you can only remove your own entry")
	ErrNotRemovable     = errors.New("entry: only completed quests can be removed")
	ErrNotToday         = errors.New("entry: you can only remove today's entry")
	ErrPhotoNotAllowed  = errors.New("entry: only a completed quest can have a photo")
	ErrPhotoNotOwn      = errors.New("entry: you can only add a photo to your own entry")
	ErrPhotoTooLate     = errors.New("entry: you can only add a photo on the same day")
	ErrPhotoExists      = errors.New("entry: this entry already has a photo")
	ErrInvalidPhoto     = errors.New("entry: image must be jpg, png or webp, up to 8MB")
)

// MaxPhotoBytes is the largest photo an entry accepts.
const MaxPhotoBytes = 8 << 20

// AllowedPhotoTypes lists the image content types an entry photo may have.
var AllowedPhotoTypes = []string{"image/jpeg", "image/png", "image/webp"}

// Entry records that a rule was scored for a member on a given day.
type Entry struct {
	ID         string
	GuildID    string
	RuleID     string         // reference only; no FK, see rule 6
	RuleName   string         // snapshot of rule.Name at creation time
	ScoreType  rule.ScoreType // snapshot of rule.ScoreType at creation time
	Points     int            // signed: +rule.Score for sum, -rule.Score for decrease
	MemberID   string         // whose score this entry affects
	LoggedBy   string         // who recorded it; equals MemberID for a self-checked quest
	OccurredOn time.Time      // date-only (UTC midnight), the calendar day it happened
	PeriodKey  time.Time      // date-only; OccurredOn for a daily rule, that week's Monday for a weekly rule
	CreatedAt  time.Time
	// PhotoURL is the delivery URL of the proof photo; empty when there is none.
	PhotoURL string
	// PhotoPublicID is the storage-side id of the photo; empty when there is none.
	PhotoPublicID string
}

// New builds a validated Entry. now must already be the caller's current wall-clock
// time *converted into the guild's timezone* (see Architecture) — New only reads its
// date components, it never consults a clock or a location itself.
func New(id, guildID, ruleID, ruleName string, scoreType rule.ScoreType, score int, memberID, loggedBy string, frequency rule.Frequency, now time.Time) (Entry, error) {
	if scoreType == rule.ScoreTypeSum && memberID != loggedBy {
		return Entry{}, ErrWrongMember
	}
	points := score
	if scoreType == rule.ScoreTypeDecrease {
		points = -score
	}
	occurredOn := DateOnly(now)
	periodKey := occurredOn
	if frequency == rule.FrequencyWeekly {
		periodKey = WeekStart(occurredOn)
	}
	return Entry{
		ID: id, GuildID: guildID, RuleID: ruleID, RuleName: ruleName, ScoreType: scoreType, Points: points,
		MemberID: memberID, LoggedBy: loggedBy, OccurredOn: occurredOn, PeriodKey: periodKey,
		CreatedAt: now.Truncate(time.Microsecond),
	}, nil
}

// DateOnly keeps only now's year/month/day, read in now's own location.
func DateOnly(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// WeekStart returns the Monday on/before d (d must already be date-only).
func WeekStart(d time.Time) time.Time {
	offset := (int(d.Weekday()) + 6) % 7 // Monday -> 0 ... Sunday -> 6
	return d.AddDate(0, 0, -offset)
}

// ValidatePhotoAttach reports whether callerID may attach a photo to e at now
// (already converted into the guild's timezone). It never mutates e.
func (e Entry) ValidatePhotoAttach(callerID string, now time.Time) error {
	if e.ScoreType != rule.ScoreTypeSum {
		return ErrPhotoNotAllowed
	}
	if e.MemberID != callerID {
		return ErrPhotoNotOwn
	}
	if e.PhotoURL != "" {
		return ErrPhotoExists
	}
	if !e.OccurredOn.Equal(DateOnly(now)) {
		return ErrPhotoTooLate
	}
	return nil
}

// ValidPhotoFile reports whether contentType and size are acceptable for a photo.
func ValidPhotoFile(contentType string, size int64) bool {
	return slices.Contains(AllowedPhotoTypes, contentType) && size > 0 && size <= MaxPhotoBytes
}
