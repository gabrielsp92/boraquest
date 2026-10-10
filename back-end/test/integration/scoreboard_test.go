package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

func TestScoreboardZeroFillAndAggregation(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	betoToken := login(t, srv.URL, "beto@boraquest.dev")
	sumRule := createRule(t, srv.URL, liaToken, dailySumRuleReq)
	decreaseRule := createRule(t, srv.URL, liaToken, decreaseRuleReq)

	// Lia completes a quest (+10, 1 completion).
	res := call(t, http.MethodPost, srv.URL+"/api/v1/entries", liaToken, controllers.EntryRequest{RuleID: sumRule.ID, MemberID: "lia"})
	require.Equal(t, http.StatusCreated, res.StatusCode)

	// Beto logs a slip against Lia (-15, no completion).
	res = call(t, http.MethodPost, srv.URL+"/api/v1/entries", betoToken, controllers.EntryRequest{RuleID: decreaseRule.ID, MemberID: "lia"})
	require.Equal(t, http.StatusCreated, res.StatusCode)

	res = call(t, http.MethodGet, srv.URL+"/api/v1/scoreboard?period=week", liaToken, nil)
	require.Equal(t, http.StatusOK, res.StatusCode)
	body := decodeBody[controllers.ScoreboardResponse](t, res)

	require.Len(t, body.Standings, 4)
	byID := make(map[string]controllers.StandingResponse, len(body.Standings))
	for _, s := range body.Standings {
		byID[s.MemberID] = s
	}
	assert.Equal(t, controllers.StandingResponse{MemberID: "lia", Points: -5, Completed: 1}, byID["lia"])
	assert.Equal(t, controllers.StandingResponse{MemberID: "beto", Points: 0, Completed: 0}, byID["beto"])
	assert.Equal(t, controllers.StandingResponse{MemberID: "nena", Points: 0, Completed: 0}, byID["nena"])
	assert.Equal(t, controllers.StandingResponse{MemberID: "caio", Points: 0, Completed: 0}, byID["caio"])

	// Lia's -5 is last; the three zero-scored members tie-break by member id ascending.
	ids := make([]string, len(body.Standings))
	for i, s := range body.Standings {
		ids[i] = s.MemberID
	}
	assert.Equal(t, []string{"beto", "caio", "nena", "lia"}, ids)

	// Same entries are still within the current month, so the monthly totals match.
	res = call(t, http.MethodGet, srv.URL+"/api/v1/scoreboard?period=month", liaToken, nil)
	require.Equal(t, http.StatusOK, res.StatusCode)
	monthBody := decodeBody[controllers.ScoreboardResponse](t, res)
	monthByID := make(map[string]controllers.StandingResponse, len(monthBody.Standings))
	for _, s := range monthBody.Standings {
		monthByID[s.MemberID] = s
	}
	assert.Equal(t, controllers.StandingResponse{MemberID: "lia", Points: -5, Completed: 1}, monthByID["lia"])
}

// TestScoreboardSurvivesRuleDeletion verifies the scoreboard aggregates raw
// entries and never joins back to rules (Business rule 6 from quest-entries.md,
// reused here).
func TestScoreboardSurvivesRuleDeletion(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	r := createRule(t, srv.URL, liaToken, dailySumRuleReq)
	res := call(t, http.MethodPost, srv.URL+"/api/v1/entries", liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "lia"})
	require.Equal(t, http.StatusCreated, res.StatusCode)

	res = call(t, http.MethodDelete, srv.URL+"/api/v1/rules/"+r.ID, liaToken, nil)
	require.Equal(t, http.StatusNoContent, res.StatusCode)

	res = call(t, http.MethodGet, srv.URL+"/api/v1/scoreboard?period=week", liaToken, nil)
	require.Equal(t, http.StatusOK, res.StatusCode)
	body := decodeBody[controllers.ScoreboardResponse](t, res)
	for _, s := range body.Standings {
		if s.MemberID == "lia" {
			assert.Equal(t, 10, s.Points)
			assert.Equal(t, 1, s.Completed)
		}
	}
}

func TestScoreboardRequiresValidPeriod(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")

	for _, path := range []string{"/api/v1/scoreboard", "/api/v1/scoreboard?period=", "/api/v1/scoreboard?period=today"} {
		res := call(t, http.MethodGet, srv.URL+path, liaToken, nil)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode, path)
	}
}

func TestScoreboardRequiresAuth(t *testing.T) {
	srv := newServer(t)

	for name, token := range map[string]string{"missing": "", "invalid": "garbage"} {
		t.Run(name, func(t *testing.T) {
			res := call(t, http.MethodGet, srv.URL+"/api/v1/scoreboard?period=week", token, nil)
			assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
		})
	}
}

func TestScoreboardForbidsUsersOutsideTheGuild(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	token := login(t, srv.URL, outsiderID+"@boraquest.dev")

	res := call(t, http.MethodGet, srv.URL+"/api/v1/scoreboard?period=week", token, nil)

	require.Equal(t, http.StatusForbidden, res.StatusCode)
	assert.Equal(t, "you are not a member of any guild", decodeBody[response.ErrorBody](t, res).Error)
}
