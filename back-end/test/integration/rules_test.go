package integration_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

var water = controllers.RuleRequest{Name: "Beber 2 L de água", Frequency: "daily", ScoreType: "sum", Score: 10}

func TestRulesCRUD(t *testing.T) {
	resetRules(t)
	srv := newServer(t)
	rules := srv.URL + "/api/v1/rules"
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	betoToken := login(t, srv.URL, "beto@boraquest.dev")

	// Empty guild.
	res := call(t, http.MethodGet, rules, liaToken, nil)
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, controllers.RuleListResponse{Rules: []controllers.RuleResponse{}, Limit: rule.MaxPerGuild}, decodeBody[controllers.RuleListResponse](t, res))

	// Create.
	res = call(t, http.MethodPost, rules, liaToken, water)
	require.Equal(t, http.StatusCreated, res.StatusCode)
	created := decodeBody[controllers.RuleResponse](t, res)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "lia", created.CreatedBy)
	assert.Equal(t, water.Name, created.Name)

	// Another member of the same guild sees and edits it.
	res = call(t, http.MethodGet, rules+"/"+created.ID, betoToken, nil)
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, created, decodeBody[controllers.RuleResponse](t, res))

	edit := controllers.RuleRequest{Name: "Comer fast-food", Frequency: "weekly", ScoreType: "decrease", Score: 15}
	res = call(t, http.MethodPut, rules+"/"+created.ID, betoToken, edit)
	require.Equal(t, http.StatusOK, res.StatusCode)
	updated := decodeBody[controllers.RuleResponse](t, res)
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, "lia", updated.CreatedBy, "createdBy is immutable")
	assert.Equal(t, created.CreatedAt, updated.CreatedAt)
	assert.Equal(t, controllers.RuleRequest{Name: updated.Name, Frequency: updated.Frequency, ScoreType: updated.ScoreType, Score: updated.Score}, edit)

	res = call(t, http.MethodGet, rules, liaToken, nil)
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, []controllers.RuleResponse{updated}, decodeBody[controllers.RuleListResponse](t, res).Rules)

	// Delete, then it's gone.
	res = call(t, http.MethodDelete, rules+"/"+created.ID, liaToken, nil)
	require.Equal(t, http.StatusNoContent, res.StatusCode)
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		res = call(t, method, rules+"/"+created.ID, liaToken, nil)
		assert.Equal(t, http.StatusNotFound, res.StatusCode, method)
	}
	res = call(t, http.MethodPut, rules+"/"+created.ID, liaToken, water)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestRulesRequireAuth(t *testing.T) {
	srv := newServer(t)

	for name, token := range map[string]string{"missing": "", "invalid": "garbage"} {
		t.Run(name, func(t *testing.T) {
			res := call(t, http.MethodGet, srv.URL+"/api/v1/rules", token, nil)
			assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
		})
	}
}

func TestRulesForbidUsersOutsideTheGuild(t *testing.T) {
	resetRules(t)
	srv := newServer(t)
	token := login(t, srv.URL, outsiderID+"@boraquest.dev")

	res := call(t, http.MethodPost, srv.URL+"/api/v1/rules", token, water)

	require.Equal(t, http.StatusForbidden, res.StatusCode)
	assert.Equal(t, "you are not a member of any guild", decodeBody[response.ErrorBody](t, res).Error)
	res = call(t, http.MethodGet, srv.URL+"/api/v1/rules", token, nil)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
}

func TestRulesValidation(t *testing.T) {
	resetRules(t)
	srv := newServer(t)
	token := login(t, srv.URL, "lia@boraquest.dev")

	for name, req := range map[string]controllers.RuleRequest{
		"empty name":     {Name: " ", Frequency: "daily", ScoreType: "sum", Score: 10},
		"bad frequency":  {Name: "x", Frequency: "monthly", ScoreType: "sum", Score: 10},
		"bad score type": {Name: "x", Frequency: "daily", ScoreType: "x", Score: 10},
		"score too high": {Name: "x", Frequency: "daily", ScoreType: "sum", Score: 101},
	} {
		t.Run(name, func(t *testing.T) {
			res := call(t, http.MethodPost, srv.URL+"/api/v1/rules", token, req)
			assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		})
	}
}

func TestRulesLimitPerGuild(t *testing.T) {
	resetRules(t)
	srv := newServer(t)
	token := login(t, srv.URL, "caio@boraquest.dev")

	for i := range rule.MaxPerGuild {
		res := call(t, http.MethodPost, srv.URL+"/api/v1/rules", token, controllers.RuleRequest{Name: fmt.Sprintf("Regra %d", i), Frequency: "daily", ScoreType: "sum", Score: 5})
		require.Equal(t, http.StatusCreated, res.StatusCode)
	}

	res := call(t, http.MethodPost, srv.URL+"/api/v1/rules", token, water)
	require.Equal(t, http.StatusConflict, res.StatusCode)
	assert.Equal(t, "a guild can have at most 40 rules", decodeBody[response.ErrorBody](t, res).Error)

	// Deleting one frees a slot.
	list := decodeBody[controllers.RuleListResponse](t, call(t, http.MethodGet, srv.URL+"/api/v1/rules", token, nil))
	require.Len(t, list.Rules, rule.MaxPerGuild)
	require.Equal(t, http.StatusNoContent, call(t, http.MethodDelete, srv.URL+"/api/v1/rules/"+list.Rules[0].ID, token, nil).StatusCode)
	assert.Equal(t, http.StatusCreated, call(t, http.MethodPost, srv.URL+"/api/v1/rules", token, water).StatusCode)
}

func TestRulesLimitHoldsUnderConcurrency(t *testing.T) {
	resetRules(t)
	srv := newServer(t)
	token := login(t, srv.URL, "lia@boraquest.dev")

	const attempts = rule.MaxPerGuild + 20
	statuses := make([]int, attempts)
	var wg sync.WaitGroup
	for i := range attempts {
		wg.Go(func() {
			res := call(t, http.MethodPost, srv.URL+"/api/v1/rules", token, water)
			statuses[i] = res.StatusCode
		})
	}
	wg.Wait()

	counts := map[int]int{}
	for _, s := range statuses {
		counts[s]++
	}
	assert.Equal(t, map[int]int{http.StatusCreated: rule.MaxPerGuild, http.StatusConflict: attempts - rule.MaxPerGuild}, counts)
	list := decodeBody[controllers.RuleListResponse](t, call(t, http.MethodGet, srv.URL+"/api/v1/rules", token, nil))
	assert.Len(t, list.Rules, rule.MaxPerGuild)
}
