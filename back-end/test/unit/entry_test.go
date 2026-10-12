package unit_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
)

func TestNewEntrySum(t *testing.T) {
	at := time.Date(2026, 10, 8, 12, 0, 0, 123456789, time.UTC)

	e, err := entry.New("e1", "g1", "r1", "Beber água", rule.ScoreTypeSum, 10, "lia", "lia", rule.FrequencyDaily, at)

	require.NoError(t, err)
	wantDay := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, entry.Entry{
		ID: "e1", GuildID: "g1", RuleID: "r1", RuleName: "Beber água", ScoreType: rule.ScoreTypeSum,
		Points: 10, MemberID: "lia", LoggedBy: "lia", OccurredOn: wantDay, PeriodKey: wantDay,
		CreatedAt: at.Truncate(time.Microsecond),
	}, e)
}

func TestNewEntryDecreaseNegatesPoints(t *testing.T) {
	at := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

	e, err := entry.New("e1", "g1", "r1", "Fast-food", rule.ScoreTypeDecrease, 15, "beto", "lia", rule.FrequencyDaily, at)

	require.NoError(t, err)
	assert.Equal(t, -15, e.Points)
	assert.Equal(t, "beto", e.MemberID)
	assert.Equal(t, "lia", e.LoggedBy)
}

func TestNewEntryDecreaseAllowsDifferentLogger(t *testing.T) {
	_, err := entry.New("e1", "g1", "r1", "Fast-food", rule.ScoreTypeDecrease, 15, "beto", "lia", rule.FrequencyWeekly, time.Now())
	assert.NoError(t, err)
}

func TestNewEntrySumWrongMember(t *testing.T) {
	_, err := entry.New("e1", "g1", "r1", "Água", rule.ScoreTypeSum, 10, "beto", "lia", rule.FrequencyDaily, time.Now())
	assert.ErrorIs(t, err, entry.ErrWrongMember)
}

func TestNewEntryPeriodKeyDaily(t *testing.T) {
	at := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC) // Thursday

	e, err := entry.New("e1", "g1", "r1", "Água", rule.ScoreTypeSum, 10, "lia", "lia", rule.FrequencyDaily, at)

	require.NoError(t, err)
	assert.Equal(t, entry.DateOnly(at), e.PeriodKey)
}

func TestNewEntryPeriodKeyWeekly(t *testing.T) {
	at := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC) // Thursday

	e, err := entry.New("e1", "g1", "r1", "Água", rule.ScoreTypeSum, 10, "lia", "lia", rule.FrequencyWeekly, at)

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), e.PeriodKey) // Monday
}

func TestDateOnlyKeepsOnlyDate(t *testing.T) {
	at := time.Date(2026, 10, 8, 23, 59, 59, 999999999, time.UTC)
	assert.Equal(t, time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC), entry.DateOnly(at))
}

func TestWeekStart(t *testing.T) {
	cases := map[string]struct {
		day  time.Time
		want time.Time
	}{
		"monday":             {time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)},
		"sunday":             {time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)},
		"thursday":           {time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)},
		"crosses month/year": {time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 28, 0, 0, 0, 0, time.UTC)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, entry.WeekStart(tc.day))
		})
	}
}

func TestValidatePhotoAttach(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	today := entry.DateOnly(now)
	ok := entry.Entry{ScoreType: rule.ScoreTypeSum, MemberID: "lia", OccurredOn: today}

	cases := map[string]struct {
		mutate func(e *entry.Entry)
		caller string
		want   error
	}{
		"own sum entry today":  {func(*entry.Entry) {}, "lia", nil},
		"decrease entry":       {func(e *entry.Entry) { e.ScoreType = rule.ScoreTypeDecrease }, "lia", entry.ErrPhotoNotAllowed},
		"someone else's entry": {func(*entry.Entry) {}, "beto", entry.ErrPhotoNotOwn},
		"already has a photo":  {func(e *entry.Entry) { e.PhotoURL = "https://x/y.jpg" }, "lia", entry.ErrPhotoExists},
		"yesterday's entry":    {func(e *entry.Entry) { e.OccurredOn = today.AddDate(0, 0, -1) }, "lia", entry.ErrPhotoTooLate},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := ok
			tc.mutate(&e)
			err := e.ValidatePhotoAttach(tc.caller, now)
			if tc.want == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.want)
			}
		})
	}
}

func TestValidPhotoFile(t *testing.T) {
	for _, ct := range entry.AllowedPhotoTypes {
		assert.True(t, entry.ValidPhotoFile(ct, 1), ct)
		assert.True(t, entry.ValidPhotoFile(ct, entry.MaxPhotoBytes), ct+" at the size limit")
		assert.False(t, entry.ValidPhotoFile(ct, entry.MaxPhotoBytes+1), ct+" one byte over")
		assert.False(t, entry.ValidPhotoFile(ct, 0), ct+" empty")
		assert.False(t, entry.ValidPhotoFile(ct, -1), ct+" negative")
	}
	assert.False(t, entry.ValidPhotoFile("image/gif", 10))
	assert.False(t, entry.ValidPhotoFile("text/plain; charset=utf-8", 10))
}
