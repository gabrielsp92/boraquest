// Package rule holds the domain model for guild rules (quests).
package rule

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// Frequency is how often a rule can be scored.
type Frequency string

const (
	FrequencyDaily  Frequency = "daily"
	FrequencyWeekly Frequency = "weekly"
)

// ScoreType says whether completing a rule adds or removes points.
type ScoreType string

const (
	ScoreTypeSum      ScoreType = "sum"
	ScoreTypeDecrease ScoreType = "decrease"
)

const (
	// MaxPerGuild caps how many rules a guild may hold at the same time.
	MaxPerGuild = 40
	MinScore    = 1
	MaxScore    = 100
	MaxNameLen  = 32
)

var (
	ErrInvalidName      = errors.New("rule: name must have 1 to 32 characters")
	ErrInvalidFrequency = errors.New("rule: invalid frequency")
	ErrInvalidScoreType = errors.New("rule: invalid score type")
	ErrInvalidScore     = errors.New("rule: score must be between 1 and 100")
	ErrNotFound         = errors.New("rule: not found")
	ErrLimitReached     = errors.New("rule: guild rule limit reached")
)

// Validate reports whether f is a known frequency.
func (f Frequency) Validate() error {
	switch f {
	case FrequencyDaily, FrequencyWeekly:
		return nil
	default:
		return ErrInvalidFrequency
	}
}

// Validate reports whether t is a known score type.
func (t ScoreType) Validate() error {
	switch t {
	case ScoreTypeSum, ScoreTypeDecrease:
		return nil
	default:
		return ErrInvalidScoreType
	}
}

// Rule is a quest a guild's members score against.
type Rule struct {
	ID        string
	GuildID   string
	Name      string
	Frequency Frequency
	ScoreType ScoreType
	Score     int
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// New builds a validated Rule.
func New(id, guildID, createdBy, name string, frequency Frequency, scoreType ScoreType, score int, now time.Time) (Rule, error) {
	now = now.Truncate(time.Microsecond)
	r := Rule{ID: id, GuildID: guildID, CreatedBy: createdBy, CreatedAt: now}
	if err := r.Update(name, frequency, scoreType, score, now); err != nil {
		return Rule{}, err
	}
	return r, nil
}

// Update replaces the editable fields of r after validating them.
// r is left untouched when validation fails. Timestamps are kept at
// microsecond precision so they round-trip through storage unchanged.
func (r *Rule) Update(name string, frequency Frequency, scoreType ScoreType, score int, now time.Time) error {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n == 0 || n > MaxNameLen {
		return ErrInvalidName
	}
	if err := frequency.Validate(); err != nil {
		return err
	}
	if err := scoreType.Validate(); err != nil {
		return err
	}
	if score < MinScore || score > MaxScore {
		return ErrInvalidScore
	}
	r.Name, r.Frequency, r.ScoreType, r.Score, r.UpdatedAt = name, frequency, scoreType, score, now.Truncate(time.Microsecond)
	return nil
}
