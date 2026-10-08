package unit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// fakeRuleUseCase returns rule/err for every call and records what it received.
type fakeRuleUseCase struct {
	rules  []rule.Rule
	rule   rule.Rule
	err    error
	userID string
	id     string
	in     service.RuleInput
}

func (f *fakeRuleUseCase) List(_ context.Context, userID string) ([]rule.Rule, error) {
	f.userID = userID
	return f.rules, f.err
}

func (f *fakeRuleUseCase) Get(_ context.Context, userID, id string) (rule.Rule, error) {
	f.userID, f.id = userID, id
	return f.rule, f.err
}

func (f *fakeRuleUseCase) Create(_ context.Context, userID string, in service.RuleInput) (rule.Rule, error) {
	f.userID, f.in = userID, in
	return f.rule, f.err
}

func (f *fakeRuleUseCase) Update(_ context.Context, userID, id string, in service.RuleInput) (rule.Rule, error) {
	f.userID, f.id, f.in = userID, id, in
	return f.rule, f.err
}

func (f *fakeRuleUseCase) Delete(_ context.Context, userID, id string) error {
	f.userID, f.id = userID, id
	return f.err
}

var sampleRule = rule.Rule{
	ID: "r1", GuildID: "g1", Name: "Água", Frequency: rule.FrequencyDaily, ScoreType: rule.ScoreTypeSum,
	Score: 10, CreatedBy: "lia", CreatedAt: ruleAt, UpdatedAt: ruleAt,
}

var sampleResponse = controllers.RuleResponse{
	ID: "r1", Name: "Água", Frequency: "daily", ScoreType: "sum", Score: 10, CreatedBy: "lia", CreatedAt: ruleAt, UpdatedAt: ruleAt,
}

func serveRules(t *testing.T, uc controllers.RuleUseCase, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := controllers.NewRuleController(uc)
	r := gin.New()
	g := r.Group("/rules", middleware.RequireAuth(fakeVerifier{userID: "lia"}))
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/:id", h.Get)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &v))
	return v
}

const validRuleBody = `{"name":"Água","frequency":"daily","scoreType":"sum","score":10}`

func TestRuleControllerList(t *testing.T) {
	uc := &fakeRuleUseCase{rules: []rule.Rule{sampleRule}}

	rec := serveRules(t, uc, http.MethodGet, "/rules", "")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, controllers.RuleListResponse{Rules: []controllers.RuleResponse{sampleResponse}, Limit: rule.MaxPerGuild}, decode[controllers.RuleListResponse](t, rec))
	assert.Equal(t, "lia", uc.userID)
}

func TestRuleControllerListEmptyIsArray(t *testing.T) {
	rec := serveRules(t, &fakeRuleUseCase{}, http.MethodGet, "/rules", "")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"rules":[],"limit":40}`, rec.Body.String())
}

func TestRuleControllerGet(t *testing.T) {
	uc := &fakeRuleUseCase{rule: sampleRule}

	rec := serveRules(t, uc, http.MethodGet, "/rules/r1", "")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, sampleResponse, decode[controllers.RuleResponse](t, rec))
	assert.Equal(t, "r1", uc.id)
}

func TestRuleControllerCreate(t *testing.T) {
	uc := &fakeRuleUseCase{rule: sampleRule}

	rec := serveRules(t, uc, http.MethodPost, "/rules", validRuleBody)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, sampleResponse, decode[controllers.RuleResponse](t, rec))
	assert.Equal(t, service.RuleInput{Name: "Água", Frequency: rule.FrequencyDaily, ScoreType: rule.ScoreTypeSum, Score: 10}, uc.in)
	assert.Equal(t, "lia", uc.userID)
}

func TestRuleControllerUpdate(t *testing.T) {
	uc := &fakeRuleUseCase{rule: sampleRule}

	rec := serveRules(t, uc, http.MethodPut, "/rules/r1", `{"name":"Fast-food","frequency":"weekly","scoreType":"decrease","score":5}`)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, sampleResponse, decode[controllers.RuleResponse](t, rec))
	assert.Equal(t, "r1", uc.id)
	assert.Equal(t, service.RuleInput{Name: "Fast-food", Frequency: rule.FrequencyWeekly, ScoreType: rule.ScoreTypeDecrease, Score: 5}, uc.in)
}

func TestRuleControllerDelete(t *testing.T) {
	uc := &fakeRuleUseCase{}

	rec := serveRules(t, uc, http.MethodDelete, "/rules/r1", "")

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
	assert.Equal(t, "r1", uc.id)
}

func TestRuleControllerInvalidBody(t *testing.T) {
	for _, body := range []string{`not json`, `{"score":1.5}`} {
		for _, call := range [][2]string{{http.MethodPost, "/rules"}, {http.MethodPut, "/rules/r1"}} {
			t.Run(call[0]+" "+body, func(t *testing.T) {
				uc := &fakeRuleUseCase{}
				rec := serveRules(t, uc, call[0], call[1], body)

				require.Equal(t, http.StatusBadRequest, rec.Code)
				assert.Equal(t, "invalid request body", decode[response.ErrorBody](t, rec).Error)
				assert.Empty(t, uc.userID, "use-case must not be called")
			})
		}
	}
}

func TestRuleControllerErrorMapping(t *testing.T) {
	cases := []struct {
		err     error
		status  int
		message string
	}{
		{rule.ErrInvalidName, http.StatusBadRequest, rule.ErrInvalidName.Error()},
		{rule.ErrInvalidFrequency, http.StatusBadRequest, rule.ErrInvalidFrequency.Error()},
		{rule.ErrInvalidScoreType, http.StatusBadRequest, rule.ErrInvalidScoreType.Error()},
		{rule.ErrInvalidScore, http.StatusBadRequest, rule.ErrInvalidScore.Error()},
		{guild.ErrNotMember, http.StatusForbidden, "you are not a member of any guild"},
		{rule.ErrNotFound, http.StatusNotFound, "rule not found"},
		{rule.ErrLimitReached, http.StatusConflict, "a guild can have at most 40 rules"},
		{errBoom, http.StatusInternalServerError, "internal error"},
	}
	calls := [][3]string{
		{http.MethodGet, "/rules", ""},
		{http.MethodGet, "/rules/r1", ""},
		{http.MethodPost, "/rules", validRuleBody},
		{http.MethodPut, "/rules/r1", validRuleBody},
		{http.MethodDelete, "/rules/r1", ""},
	}
	for _, tc := range cases {
		for _, call := range calls {
			t.Run(fmt.Sprintf("%s %s %v", call[0], call[1], tc.err), func(t *testing.T) {
				rec := serveRules(t, &fakeRuleUseCase{err: tc.err}, call[0], call[1], call[2])

				require.Equal(t, tc.status, rec.Code)
				assert.Equal(t, tc.message, decode[response.ErrorBody](t, rec).Error)
			})
		}
	}
}
