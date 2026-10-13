package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/weekresult"
)

func TestWeekResultWinnerIsFirstStanding(t *testing.T) {
	w := weekresult.WeekResult{Standings: []weekresult.Standing{
		{MemberID: "nena", Rank: 1, Points: 40},
		{MemberID: "lia", Rank: 2, Points: 10},
	}}

	winner, ok := w.Winner()

	assert.True(t, ok)
	assert.Equal(t, "nena", winner.MemberID)
}

func TestWeekResultWinnerWithoutStandings(t *testing.T) {
	winner, ok := weekresult.WeekResult{}.Winner()

	assert.False(t, ok)
	assert.Equal(t, weekresult.Standing{}, winner)
}
