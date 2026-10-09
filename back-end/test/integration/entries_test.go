package integration_test

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

var dailySumRuleReq = controllers.RuleRequest{Name: "Beber 2 L de água", Frequency: "daily", ScoreType: "sum", Score: 10}
var weeklySumRuleReq = controllers.RuleRequest{Name: "Treino", Frequency: "weekly", ScoreType: "sum", Score: 20}
var decreaseRuleReq = controllers.RuleRequest{Name: "Fast-food", Frequency: "daily", ScoreType: "decrease", Score: 15}

func createRule(t *testing.T, baseURL, token string, req controllers.RuleRequest) controllers.RuleResponse {
	t.Helper()
	res := call(t, http.MethodPost, baseURL+"/api/v1/rules", token, req)
	require.Equal(t, http.StatusCreated, res.StatusCode)
	return decodeBody[controllers.RuleResponse](t, res)
}

func TestEntriesSumChecklistCRUD(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	r := createRule(t, srv.URL, liaToken, dailySumRuleReq)
	entries := srv.URL + "/api/v1/entries"

	// Create.
	res := call(t, http.MethodPost, entries, liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "lia"})
	require.Equal(t, http.StatusCreated, res.StatusCode)
	created := decodeBody[controllers.EntryResponse](t, res)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, r.ID, created.RuleID)
	assert.Equal(t, r.Name, created.RuleName)
	assert.Equal(t, "sum", created.ScoreType)
	assert.Equal(t, 10, created.Points)
	assert.Equal(t, "lia", created.MemberID)
	assert.Equal(t, "lia", created.LoggedBy)

	// Shows up on today's list.
	res = call(t, http.MethodGet, entries+"?period=today", liaToken, nil)
	require.Equal(t, http.StatusOK, res.StatusCode)
	list := decodeBody[controllers.EntryListResponse](t, res)
	assert.Equal(t, []controllers.EntryResponse{created}, list.Entries)
	assert.Equal(t, list.Today, list.From)
	assert.Equal(t, list.Today, list.To)

	// Also shows up on this week's list.
	res = call(t, http.MethodGet, entries+"?period=week", liaToken, nil)
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, []controllers.EntryResponse{created}, decodeBody[controllers.EntryListResponse](t, res).Entries)

	// Duplicate for the same rule/member/period is rejected.
	res = call(t, http.MethodPost, entries, liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "lia"})
	require.Equal(t, http.StatusConflict, res.StatusCode)
	assert.Equal(t, "entry: already logged for this period", decodeBody[response.ErrorBody](t, res).Error)

	// Remove, then it's gone.
	res = call(t, http.MethodDelete, entries+"/"+created.ID, liaToken, nil)
	require.Equal(t, http.StatusNoContent, res.StatusCode)
	res = call(t, http.MethodGet, entries+"?period=today", liaToken, nil)
	assert.Empty(t, decodeBody[controllers.EntryListResponse](t, res).Entries)

	// Double-delete is a 404.
	res = call(t, http.MethodDelete, entries+"/"+created.ID, liaToken, nil)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)

	// And it can be logged again now that it was removed.
	res = call(t, http.MethodPost, entries, liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "lia"})
	assert.Equal(t, http.StatusCreated, res.StatusCode)
}

func TestEntriesSumRejectsOtherMember(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	r := createRule(t, srv.URL, liaToken, dailySumRuleReq)

	res := call(t, http.MethodPost, srv.URL+"/api/v1/entries", liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "beto"})

	require.Equal(t, http.StatusForbidden, res.StatusCode)
	assert.Equal(t, "entry: you can only complete a quest for yourself", decodeBody[response.ErrorBody](t, res).Error)
}

func TestEntriesWeeklySumUniquePerWeek(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	r := createRule(t, srv.URL, liaToken, weeklySumRuleReq)

	res := call(t, http.MethodPost, srv.URL+"/api/v1/entries", liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "lia"})
	require.Equal(t, http.StatusCreated, res.StatusCode)

	res = call(t, http.MethodPost, srv.URL+"/api/v1/entries", liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "lia"})
	assert.Equal(t, http.StatusConflict, res.StatusCode)
}

func TestEntriesDecreaseAnyMemberRepeatable(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	r := createRule(t, srv.URL, liaToken, decreaseRuleReq)
	entries := srv.URL + "/api/v1/entries"

	// Log for self.
	res := call(t, http.MethodPost, entries, liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "lia"})
	require.Equal(t, http.StatusCreated, res.StatusCode)
	self := decodeBody[controllers.EntryResponse](t, res)
	assert.Equal(t, -15, self.Points)

	// Log for someone else.
	res = call(t, http.MethodPost, entries, liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "beto"})
	require.Equal(t, http.StatusCreated, res.StatusCode)
	other := decodeBody[controllers.EntryResponse](t, res)
	assert.Equal(t, "beto", other.MemberID)
	assert.Equal(t, "lia", other.LoggedBy)

	// Repeat slip for the same member/rule/day is allowed twice.
	res = call(t, http.MethodPost, entries, liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "beto"})
	require.Equal(t, http.StatusCreated, res.StatusCode)
	assert.NotEqual(t, other.ID, decodeBody[controllers.EntryResponse](t, res).ID)

	// Never removable.
	res = call(t, http.MethodDelete, entries+"/"+self.ID, liaToken, nil)
	require.Equal(t, http.StatusForbidden, res.StatusCode)
	assert.Equal(t, "entry: only completed quests can be removed", decodeBody[response.ErrorBody](t, res).Error)
}

func TestEntriesMemberMustBeInGuild(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	r := createRule(t, srv.URL, liaToken, decreaseRuleReq)

	res := call(t, http.MethodPost, srv.URL+"/api/v1/entries", liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: outsiderID})

	require.Equal(t, http.StatusBadRequest, res.StatusCode)
	assert.Equal(t, "entry: member is not in your guild", decodeBody[response.ErrorBody](t, res).Error)
}

func TestEntriesRuleNotFound(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")

	res := call(t, http.MethodPost, srv.URL+"/api/v1/entries", liaToken, controllers.EntryRequest{RuleID: "ghost", MemberID: "lia"})

	require.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Equal(t, "rule not found", decodeBody[response.ErrorBody](t, res).Error)
}

func TestEntriesDeleteRejectsOthersEntry(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	betoToken := login(t, srv.URL, "beto@boraquest.dev")
	r := createRule(t, srv.URL, liaToken, dailySumRuleReq)
	res := call(t, http.MethodPost, srv.URL+"/api/v1/entries", liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "lia"})
	require.Equal(t, http.StatusCreated, res.StatusCode)
	created := decodeBody[controllers.EntryResponse](t, res)

	res = call(t, http.MethodDelete, srv.URL+"/api/v1/entries/"+created.ID, betoToken, nil)

	require.Equal(t, http.StatusForbidden, res.StatusCode)
	assert.Equal(t, "entry: you can only remove your own entry", decodeBody[response.ErrorBody](t, res).Error)
}

func TestEntriesDeleteRejectsPastDay(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	r := createRule(t, srv.URL, liaToken, dailySumRuleReq)
	res := call(t, http.MethodPost, srv.URL+"/api/v1/entries", liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "lia"})
	require.Equal(t, http.StatusCreated, res.StatusCode)
	created := decodeBody[controllers.EntryResponse](t, res)

	// Simulate the entry having happened yesterday.
	_, err := pool.Exec(context.Background(), `UPDATE entries SET occurred_on = occurred_on - INTERVAL '1 day' WHERE id = $1`, created.ID)
	require.NoError(t, err)

	res = call(t, http.MethodDelete, srv.URL+"/api/v1/entries/"+created.ID, liaToken, nil)

	require.Equal(t, http.StatusForbidden, res.StatusCode)
	assert.Equal(t, "entry: you can only remove today's entry", decodeBody[response.ErrorBody](t, res).Error)
}

func TestEntriesListRequiresValidPeriod(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")

	for _, path := range []string{"/api/v1/entries", "/api/v1/entries?period=", "/api/v1/entries?period=month"} {
		res := call(t, http.MethodGet, srv.URL+path, liaToken, nil)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode, path)
	}
}

func TestEntriesRequireAuth(t *testing.T) {
	srv := newServer(t)

	for name, token := range map[string]string{"missing": "", "invalid": "garbage"} {
		t.Run(name, func(t *testing.T) {
			res := call(t, http.MethodGet, srv.URL+"/api/v1/entries?period=today", token, nil)
			assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
		})
	}
}

func TestEntriesForbidUsersOutsideTheGuild(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	token := login(t, srv.URL, outsiderID+"@boraquest.dev")

	res := call(t, http.MethodGet, srv.URL+"/api/v1/entries?period=today", token, nil)

	require.Equal(t, http.StatusForbidden, res.StatusCode)
	assert.Equal(t, "you are not a member of any guild", decodeBody[response.ErrorBody](t, res).Error)
}

// TestEntriesSurviveRuleDeletion verifies business rule 6: deleting a rule
// always succeeds and never touches already-logged entries' snapshot data.
func TestEntriesSurviveRuleDeletion(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	r := createRule(t, srv.URL, liaToken, dailySumRuleReq)
	res := call(t, http.MethodPost, srv.URL+"/api/v1/entries", liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "lia"})
	require.Equal(t, http.StatusCreated, res.StatusCode)
	created := decodeBody[controllers.EntryResponse](t, res)

	res = call(t, http.MethodDelete, srv.URL+"/api/v1/rules/"+r.ID, liaToken, nil)
	require.Equal(t, http.StatusNoContent, res.StatusCode)

	res = call(t, http.MethodGet, srv.URL+"/api/v1/entries?period=today", liaToken, nil)
	require.Equal(t, http.StatusOK, res.StatusCode)
	list := decodeBody[controllers.EntryListResponse](t, res).Entries
	require.Len(t, list, 1)
	assert.Equal(t, created, list[0], "entry keeps its own snapshot after the rule is gone")
}

func TestEntriesSumUniqueHoldsUnderConcurrency(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	r := createRule(t, srv.URL, liaToken, dailySumRuleReq)

	const attempts = 20
	statuses := make([]int, attempts)
	var wg sync.WaitGroup
	for i := range attempts {
		wg.Go(func() {
			res := call(t, http.MethodPost, srv.URL+"/api/v1/entries", liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "lia"})
			statuses[i] = res.StatusCode
		})
	}
	wg.Wait()

	counts := map[int]int{}
	for _, s := range statuses {
		counts[s]++
	}
	assert.Equal(t, map[int]int{http.StatusCreated: 1, http.StatusConflict: attempts - 1}, counts)
}
