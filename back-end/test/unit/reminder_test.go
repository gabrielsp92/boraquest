package unit_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/reminder"
)

func TestReminderDaily(t *testing.T) {
	assert.Equal(t, reminder.Payload{
		Title: "Bora fechar o dia?",
		Body:  "Marca as quests que você fez hoje.",
		URL:   "/hoje",
		Tag:   "daily-reminder",
	}, reminder.Daily())
}

func TestReminderInWindow(t *testing.T) {
	at := func(h, m, s int) time.Time { return time.Date(2026, 10, 13, h, m, s, 0, saoPaulo) }

	assert.False(t, reminder.InWindow(at(19, 59, 59)))
	assert.True(t, reminder.InWindow(at(20, 0, 0)))
	assert.True(t, reminder.InWindow(at(22, 59, 59)))
	assert.False(t, reminder.InWindow(at(23, 0, 0)))
}

func TestReminderNeedsReminder(t *testing.T) {
	assert.False(t, reminder.NeedsReminder(nil, nil), "no daily quests")
	assert.False(t, reminder.NeedsReminder([]string{"a", "b"}, []string{"b", "a"}), "all done")
	assert.True(t, reminder.NeedsReminder([]string{"a", "b"}, []string{"a"}), "one missing")
	assert.True(t, reminder.NeedsReminder([]string{"a"}, []string{"x", "y"}), "done ids not in the list")
}
