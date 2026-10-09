// Package prize holds the domain model for a guild's week/month prizes.
package prize

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// Period selects which of a guild's two prize slots is being read or written.
type Period string

const (
	PeriodWeek  Period = "week"
	PeriodMonth Period = "month"
)

// MaxLen is the maximum number of runes a prize's text may hold once trimmed.
const MaxLen = 60

// ErrTooLong is returned when a prize's trimmed text exceeds MaxLen.
var ErrTooLong = errors.New("prize: must be at most 60 characters")

// Prizes holds both of a guild's prize slots. Either may be empty.
type Prizes struct {
	GuildID   string
	Week      string
	Month     string
	UpdatedAt time.Time
}

// Clean trims text and validates its length. Empty is always valid.
func Clean(text string) (string, error) {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > MaxLen {
		return "", ErrTooLong
	}
	return text, nil
}
