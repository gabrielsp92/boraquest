package unit_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
)

func TestFrequencyValidate(t *testing.T) {
	assert.NoError(t, rule.FrequencyDaily.Validate())
	assert.NoError(t, rule.FrequencyWeekly.Validate())
	assert.ErrorIs(t, rule.Frequency("monthly").Validate(), rule.ErrInvalidFrequency)
}

func TestScoreTypeValidate(t *testing.T) {
	assert.NoError(t, rule.ScoreTypeSum.Validate())
	assert.NoError(t, rule.ScoreTypeDecrease.Validate())
	assert.ErrorIs(t, rule.ScoreType("multiply").Validate(), rule.ErrInvalidScoreType)
}

func TestNewRule(t *testing.T) {
	at := time.Date(2026, 10, 8, 12, 0, 0, 123456789, time.UTC)

	r, err := rule.New("r1", "g1", "lia", "  Beber água  ", rule.FrequencyDaily, rule.ScoreTypeSum, 10, at)

	require.NoError(t, err)
	want := at.Truncate(time.Microsecond)
	assert.Equal(t, rule.Rule{
		ID: "r1", GuildID: "g1", Name: "Beber água", Frequency: rule.FrequencyDaily, ScoreType: rule.ScoreTypeSum,
		Score: 10, CreatedBy: "lia", CreatedAt: want, UpdatedAt: want,
	}, r)
}

func TestNewRuleValidation(t *testing.T) {
	at := time.Now()
	cases := map[string]struct {
		name      string
		frequency rule.Frequency
		scoreType rule.ScoreType
		score     int
		want      error
	}{
		"empty name":       {"   ", rule.FrequencyDaily, rule.ScoreTypeSum, 10, rule.ErrInvalidName},
		"name too long":    {strings.Repeat("á", rule.MaxNameLen+1), rule.FrequencyDaily, rule.ScoreTypeSum, 10, rule.ErrInvalidName},
		"bad frequency":    {"x", "monthly", rule.ScoreTypeSum, 10, rule.ErrInvalidFrequency},
		"bad score type":   {"x", rule.FrequencyDaily, "multiply", 10, rule.ErrInvalidScoreType},
		"score too low":    {"x", rule.FrequencyDaily, rule.ScoreTypeSum, rule.MinScore - 1, rule.ErrInvalidScore},
		"score too high":   {"x", rule.FrequencyDaily, rule.ScoreTypeSum, rule.MaxScore + 1, rule.ErrInvalidScore},
		"negative score":   {"x", rule.FrequencyWeekly, rule.ScoreTypeDecrease, -5, rule.ErrInvalidScore},
		"max length is ok": {strings.Repeat("á", rule.MaxNameLen), rule.FrequencyWeekly, rule.ScoreTypeDecrease, rule.MaxScore, nil},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := rule.New("r1", "g1", "lia", tc.name, tc.frequency, tc.scoreType, tc.score, at)
			if tc.want == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestRuleUpdate(t *testing.T) {
	created := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	r, err := rule.New("r1", "g1", "lia", "Água", rule.FrequencyDaily, rule.ScoreTypeSum, 10, created)
	require.NoError(t, err)

	require.NoError(t, r.Update("Fast-food", rule.FrequencyWeekly, rule.ScoreTypeDecrease, 20, updated))

	assert.Equal(t, "Fast-food", r.Name)
	assert.Equal(t, rule.FrequencyWeekly, r.Frequency)
	assert.Equal(t, rule.ScoreTypeDecrease, r.ScoreType)
	assert.Equal(t, 20, r.Score)
	assert.Equal(t, created, r.CreatedAt)
	assert.Equal(t, updated, r.UpdatedAt)
	assert.Equal(t, "lia", r.CreatedBy)
}

func TestRuleUpdateInvalidLeavesRuleUntouched(t *testing.T) {
	r, err := rule.New("r1", "g1", "lia", "Água", rule.FrequencyDaily, rule.ScoreTypeSum, 10, time.Now())
	require.NoError(t, err)
	before := r

	assert.ErrorIs(t, r.Update("Água", rule.FrequencyDaily, rule.ScoreTypeSum, 0, time.Now()), rule.ErrInvalidScore)
	assert.Equal(t, before, r)
}
