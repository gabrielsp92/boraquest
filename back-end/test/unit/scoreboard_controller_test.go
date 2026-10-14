package unit_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// fakeScoreboardUseCase returns standings/err for every call and records what it received.
type fakeScoreboardUseCase struct {
	standings       []service.Standing
	from, to, today time.Time
	items           []service.QuestTotal
	err             error
	userID          string
	memberID        string
	period          service.ScoreboardPeriod
}

func (f *fakeScoreboardUseCase) List(_ context.Context, userID string, period service.ScoreboardPeriod) ([]service.Standing, time.Time, time.Time, time.Time, error) {
	f.userID, f.period = userID, period
	return f.standings, f.from, f.to, f.today, f.err
}

func (f *fakeScoreboardUseCase) Breakdown(_ context.Context, userID, memberID string, period service.ScoreboardPeriod) ([]service.QuestTotal, time.Time, time.Time, time.Time, error) {
	f.userID, f.memberID, f.period = userID, memberID, period
	return f.items, f.from, f.to, f.today, f.err
}

func serveScoreboard(t *testing.T, uc controllers.ScoreboardUseCase, path string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := controllers.NewScoreboardController(uc)
	r := gin.New()
	g := r.Group("/scoreboard", middleware.RequireAuth(fakeVerifier{userID: "lia"}))
	g.GET("", h.List)
	g.GET("/members/:memberId", h.Breakdown)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer good")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestScoreboardControllerList(t *testing.T) {
	uc := &fakeScoreboardUseCase{
		standings: []service.Standing{{MemberID: "lia", Points: 30, Completed: 3}, {MemberID: "beto", Points: -15, Completed: 0}},
		from:      time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
		to:        time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC),
		today:     time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC),
	}

	rec := serveScoreboard(t, uc, "/scoreboard?period=week")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, controllers.ScoreboardResponse{
		Standings: []controllers.StandingResponse{
			{MemberID: "lia", Points: 30, Completed: 3},
			{MemberID: "beto", Points: -15, Completed: 0},
		},
		From: "2026-10-05", To: "2026-10-11", Today: "2026-10-08",
	}, decode[controllers.ScoreboardResponse](t, rec))
	assert.Equal(t, "lia", uc.userID)
	assert.Equal(t, service.ScoreboardWeek, uc.period)
}

func TestScoreboardControllerListMonth(t *testing.T) {
	uc := &fakeScoreboardUseCase{}

	rec := serveScoreboard(t, uc, "/scoreboard?period=month")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, service.ScoreboardMonth, uc.period)
}

func TestScoreboardControllerListEmptyIsArray(t *testing.T) {
	rec := serveScoreboard(t, &fakeScoreboardUseCase{}, "/scoreboard?period=week")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"standings":[],"from":"0001-01-01","to":"0001-01-01","today":"0001-01-01"}`, rec.Body.String())
}

func TestScoreboardControllerListMissingOrInvalidPeriod(t *testing.T) {
	for _, path := range []string{"/scoreboard", "/scoreboard?period=", "/scoreboard?period=today"} {
		t.Run(path, func(t *testing.T) {
			uc := &fakeScoreboardUseCase{}
			rec := serveScoreboard(t, uc, path)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Empty(t, uc.userID, "use-case must not be called")
		})
	}
}

func TestScoreboardControllerErrorMapping(t *testing.T) {
	cases := []struct {
		err     error
		status  int
		message string
	}{
		{guild.ErrNotMember, http.StatusForbidden, "you are not a member of any guild"},
		{errBoom, http.StatusInternalServerError, "internal error"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%v", tc.err), func(t *testing.T) {
			rec := serveScoreboard(t, &fakeScoreboardUseCase{err: tc.err}, "/scoreboard?period=week")

			require.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.message, decode[response.ErrorBody](t, rec).Error)
		})
	}
}

func TestScoreboardControllerBreakdown(t *testing.T) {
	uc := &fakeScoreboardUseCase{
		items: []service.QuestTotal{
			{RuleID: "r1", RuleName: "Beber 2 L de água", ScoreType: rule.ScoreTypeSum, Count: 5, Points: 50},
			{RuleID: "r2", RuleName: "Ler 20 minutos", ScoreType: rule.ScoreTypeSum, Count: 2, Points: 30},
			{RuleID: "r9", RuleName: "Esquecer a louça", ScoreType: rule.ScoreTypeDecrease, Count: 1, Points: -10},
		},
		from:  time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC),
		to:    time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC),
		today: time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC),
	}

	rec := serveScoreboard(t, uc, "/scoreboard/members/beto?period=week")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, controllers.ScoreBreakdownResponse{
		MemberID: "beto",
		Items: []controllers.QuestTotalResponse{
			{RuleID: "r1", RuleName: "Beber 2 L de água", ScoreType: "sum", Count: 5, Points: 50},
			{RuleID: "r2", RuleName: "Ler 20 minutos", ScoreType: "sum", Count: 2, Points: 30},
			{RuleID: "r9", RuleName: "Esquecer a louça", ScoreType: "decrease", Count: 1, Points: -10},
		},
		Total: 70, Completed: 7,
		From: "2026-10-12", To: "2026-10-18", Today: "2026-10-13",
	}, decode[controllers.ScoreBreakdownResponse](t, rec))
	assert.Equal(t, "lia", uc.userID)
	assert.Equal(t, "beto", uc.memberID)
	assert.Equal(t, service.ScoreboardWeek, uc.period)
}

func TestScoreboardControllerBreakdownEmptyIsArray(t *testing.T) {
	uc := &fakeScoreboardUseCase{}

	rec := serveScoreboard(t, uc, "/scoreboard/members/lia?period=month")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, service.ScoreboardMonth, uc.period)
	assert.JSONEq(t, `{"memberId":"lia","items":[],"total":0,"completed":0,"from":"0001-01-01","to":"0001-01-01","today":"0001-01-01"}`, rec.Body.String())
}

func TestScoreboardControllerBreakdownInvalidPeriod(t *testing.T) {
	uc := &fakeScoreboardUseCase{}

	rec := serveScoreboard(t, uc, "/scoreboard/members/beto?period=day")

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, uc.userID, "use-case must not be called")
}

func TestScoreboardControllerBreakdownErrorMapping(t *testing.T) {
	cases := []struct {
		err     error
		status  int
		message string
	}{
		{guild.ErrNotMember, http.StatusForbidden, "you are not a member of any guild"},
		{entry.ErrMemberNotInGuild, http.StatusNotFound, "member not found in your guild"},
		{errBoom, http.StatusInternalServerError, "internal error"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%v", tc.err), func(t *testing.T) {
			rec := serveScoreboard(t, &fakeScoreboardUseCase{err: tc.err}, "/scoreboard/members/beto?period=week")

			require.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.message, decode[response.ErrorBody](t, rec).Error)
		})
	}
}
