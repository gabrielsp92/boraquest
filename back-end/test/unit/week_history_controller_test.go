package unit_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/weekresult"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

type fakeWeekHistoryUseCase struct {
	weeks  []weekresult.WeekResult
	err    error
	userID string
}

func (f *fakeWeekHistoryUseCase) List(_ context.Context, userID string) ([]weekresult.WeekResult, error) {
	f.userID = userID
	return f.weeks, f.err
}

func serveWeekHistory(t *testing.T, uc controllers.WeekHistoryUseCase) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := controllers.NewWeekHistoryController(uc)
	r := gin.New()
	r.GET("/weeks/history", middleware.RequireAuth(fakeVerifier{userID: "lia"}), h.List)

	req := httptest.NewRequest(http.MethodGet, "/weeks/history", nil)
	req.Header.Set("Authorization", "Bearer good")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestWeekHistoryControllerList(t *testing.T) {
	uc := &fakeWeekHistoryUseCase{weeks: []weekresult.WeekResult{
		{
			WeekStart: day(2026, 9, 28), WeekEnd: day(2026, 10, 4), Prize: "Escolher o filme",
			Standings: []weekresult.Standing{
				{MemberID: "nena", Rank: 1, Points: 40, Completed: 4},
				{MemberID: "lia", Rank: 2, Points: -5, Completed: 0},
			},
		},
		{WeekStart: day(2026, 9, 21), WeekEnd: day(2026, 9, 27)}, // no standings: zero winner
	}}

	rec := serveWeekHistory(t, uc)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "lia", uc.userID)
	assert.Equal(t, controllers.WeekHistoryResponse{Weeks: []controllers.WeekResultResponse{
		{
			From: "2026-09-28", To: "2026-10-04", Prize: "Escolher o filme",
			Winner: controllers.WeekStandingResponse{MemberID: "nena", Rank: 1, Points: 40, Completed: 4},
			Standings: []controllers.WeekStandingResponse{
				{MemberID: "nena", Rank: 1, Points: 40, Completed: 4},
				{MemberID: "lia", Rank: 2, Points: -5, Completed: 0},
			},
		},
		{From: "2026-09-21", To: "2026-09-27", Standings: []controllers.WeekStandingResponse{}},
	}}, decode[controllers.WeekHistoryResponse](t, rec))
}

func TestWeekHistoryControllerListEmptyIsArray(t *testing.T) {
	rec := serveWeekHistory(t, &fakeWeekHistoryUseCase{})

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"weeks":[]}`, rec.Body.String())
}

func TestWeekHistoryControllerErrorMapping(t *testing.T) {
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
			rec := serveWeekHistory(t, &fakeWeekHistoryUseCase{err: tc.err})

			require.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.message, decode[response.ErrorBody](t, rec).Error)
		})
	}
}
