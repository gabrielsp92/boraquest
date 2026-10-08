package unit_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/seedfile"
)

func TestReadRules(t *testing.T) {
	rules, err := seedfile.ReadRules(strings.NewReader(`{"rules": [
		{"name": "Beber água", "frequency": "daily", "scoreType": "sum", "score": 10},
		{"name": "Fast-food", "frequency": "weekly", "scoreType": "decrease", "score": 5}
	]}`))

	require.NoError(t, err)
	assert.Equal(t, []service.RuleInput{
		{Name: "Beber água", Frequency: rule.FrequencyDaily, ScoreType: rule.ScoreTypeSum, Score: 10},
		{Name: "Fast-food", Frequency: rule.FrequencyWeekly, ScoreType: rule.ScoreTypeDecrease, Score: 5},
	}, rules)
}

func TestReadRulesErrors(t *testing.T) {
	_, err := seedfile.ReadRules(strings.NewReader(`not json`))
	assert.Error(t, err)

	_, err = seedfile.ReadRules(strings.NewReader(`{"rules": [{"name": "x", "points": 10}]}`))
	assert.ErrorContains(t, err, "points", "unknown fields are rejected")

	_, err = seedfile.ReadRules(strings.NewReader(`{"rules": []}`))
	assert.ErrorIs(t, err, seedfile.ErrNoRules)
}

// The committed seed file must stay readable and every rule valid.
func TestSeedsRulesFileIsValid(t *testing.T) {
	f, err := os.Open("../../seeds/rules.json")
	require.NoError(t, err)
	defer f.Close()

	rules, err := seedfile.ReadRules(f)
	require.NoError(t, err)
	assert.Len(t, rules, 8)
	for _, x := range rules {
		_, err := rule.New("id", "g1", "lia", x.Name, x.Frequency, x.ScoreType, x.Score, time.Now())
		assert.NoError(t, err, x.Name)
	}
}
