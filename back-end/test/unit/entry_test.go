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
