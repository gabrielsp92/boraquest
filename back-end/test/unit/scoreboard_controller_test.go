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
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// fakeScoreboardUseCase returns standings/err for every call and records what it received.
type fakeScoreboardUseCase struct {
	standings       []service.Standing
	from, to, today time.Time
	err             error
	userID          string
	period          service.ScoreboardPeriod
}

func (f *fakeScoreboardUseCase) List(_ context.Context, userID string, period service.ScoreboardPeriod) ([]service.Standing, time.Time, time.Time, time.Time, error) {
	f.userID, f.period = userID, period
	return f.standings, f.from, f.to, f.today, f.err
}

func serveScoreboard(t *testing.T, uc controllers.ScoreboardUseCase, path string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := controllers.NewScoreboardController(uc)
	r := gin.New()
	g := r.Group("/scoreboard", middleware.RequireAuth(fakeVerifier{userID: "lia"}))
	g.GET("", h.List)

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
