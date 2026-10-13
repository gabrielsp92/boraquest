// Package reminder holds the rules of the daily "mark your quests" reminder.
package reminder

import (
	"errors"
	"slices"
	"time"
)

// ErrOutsideWindow is returned when the daily run is triggered outside [20:00, 23:00).
var ErrOutsideWindow = errors.New("reminder: outside the daily send window")

const (
	WindowStartHour = 20 // inclusive, America/Sao_Paulo
	WindowEndHour   = 23 // exclusive
	// TTLSeconds keeps an undelivered reminder for 3h, so it never arrives the next morning.
	TTLSeconds = 10800
	// Topic lets the push service replace a pending reminder with a newer one.
	Topic = "daily-reminder"
)

// Payload is the JSON body the service worker reads.
type Payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Tag   string `json:"tag"`
}

// Daily returns the fixed daily payload. It carries no names, points or quest names.
func Daily() Payload {
	return Payload{
		Title: "Bora fechar o dia?",
		Body:  "Marca as quests que você fez hoje.",
		URL:   "/hoje",
		Tag:   Topic,
	}
}

// InWindow reports whether local (already in America/Sao_Paulo) is in [20:00, 23:00).
func InWindow(local time.Time) bool {
	h := local.Hour()
	return h >= WindowStartHour && h < WindowEndHour
}

// NeedsReminder reports whether any rule in dailySumRuleIDs is missing from doneRuleIDs.
// It is false when dailySumRuleIDs is empty.
func NeedsReminder(dailySumRuleIDs, doneRuleIDs []string) bool {
	for _, id := range dailySumRuleIDs {
		if !slices.Contains(doneRuleIDs, id) {
			return true
		}
	}
	return false
}
