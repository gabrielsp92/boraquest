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

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/prize"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// fakePrizeUseCase returns prizes/err for every call and records what it received.
type fakePrizeUseCase struct {
	prizes prize.Prizes
	err    error
	userID string
	period prize.Period
	text   string
}

func (f *fakePrizeUseCase) Get(_ context.Context, userID string) (prize.Prizes, error) {
	f.userID = userID
	return f.prizes, f.err
}

func (f *fakePrizeUseCase) Set(_ context.Context, userID string, period prize.Period, text string) (prize.Prizes, error) {
	f.userID, f.period, f.text = userID, period, text
	return f.prizes, f.err
}

var samplePrizes = prize.Prizes{GuildID: "g1", Week: "Cinema", Month: "Jantar", UpdatedAt: prizeAt}

var samplePrizeResponse = controllers.PrizeResponse{Week: "Cinema", Month: "Jantar", UpdatedAt: prizeAt}

func servePrizes(t *testing.T, uc controllers.PrizeUseCase, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := controllers.NewPrizeController(uc)
	r := gin.New()
	g := r.Group("/prizes", middleware.RequireAuth(fakeVerifier{userID: "lia"}))
	g.GET("", h.Get)
	g.PUT("/week", h.SetWeek)
	g.PUT("/month", h.SetMonth)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestPrizeControllerGet(t *testing.T) {
	uc := &fakePrizeUseCase{prizes: samplePrizes}

	rec := servePrizes(t, uc, http.MethodGet, "/prizes", "")

	require.Equal(t, http.StatusOK, rec.Code)
	var got controllers.PrizeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, samplePrizeResponse, got)
	assert.Equal(t, "lia", uc.userID)
}

func TestPrizeControllerSetWeek(t *testing.T) {
	uc := &fakePrizeUseCase{prizes: samplePrizes}

	rec := servePrizes(t, uc, http.MethodPut, "/prizes/week", `{"text":"Cinema"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	var got controllers.PrizeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, samplePrizeResponse, got)
	assert.Equal(t, prize.PeriodWeek, uc.period)
	assert.Equal(t, "Cinema", uc.text)
	assert.Equal(t, "lia", uc.userID)
}

func TestPrizeControllerSetMonth(t *testing.T) {
	uc := &fakePrizeUseCase{prizes: samplePrizes}

	rec := servePrizes(t, uc, http.MethodPut, "/prizes/month", `{"text":"Jantar"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, prize.PeriodMonth, uc.period)
	assert.Equal(t, "Jantar", uc.text)
}

func TestPrizeControllerSetMissingTextBindsToEmpty(t *testing.T) {
	uc := &fakePrizeUseCase{prizes: samplePrizes}

	rec := servePrizes(t, uc, http.MethodPut, "/prizes/week", `{}`)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "", uc.text)
}

func TestPrizeControllerSetEmptyTextIsValid(t *testing.T) {
	uc := &fakePrizeUseCase{prizes: samplePrizes}

	rec := servePrizes(t, uc, http.MethodPut, "/prizes/week", `{"text":""}`)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "", uc.text)
}

func TestPrizeControllerSetInvalidBody(t *testing.T) {
	for _, path := range []string{"/prizes/week", "/prizes/month"} {
		t.Run(path, func(t *testing.T) {
			uc := &fakePrizeUseCase{}
			rec := servePrizes(t, uc, http.MethodPut, path, `{"text":1}`)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "invalid request body", decode[response.ErrorBody](t, rec).Error)
			assert.Empty(t, uc.userID, "use-case must not be called")
		})
	}
}

func TestPrizeControllerErrorMapping(t *testing.T) {
	cases := []struct {
		err     error
		status  int
		message string
	}{
		{guild.ErrNotMember, http.StatusForbidden, "you are not a member of any guild"},
		{prize.ErrTooLong, http.StatusBadRequest, prize.ErrTooLong.Error()},
		{errBoom, http.StatusInternalServerError, "internal error"},
	}
	calls := [][3]string{
		{http.MethodGet, "/prizes", ""},
		{http.MethodPut, "/prizes/week", `{"text":"Cinema"}`},
		{http.MethodPut, "/prizes/month", `{"text":"Cinema"}`},
	}
	for _, tc := range cases {
		for _, call := range calls {
			t.Run(fmt.Sprintf("%s %s %v", call[0], call[1], tc.err), func(t *testing.T) {
				rec := servePrizes(t, &fakePrizeUseCase{err: tc.err}, call[0], call[1], call[2])

				require.Equal(t, tc.status, rec.Code)
				assert.Equal(t, tc.message, decode[response.ErrorBody](t, rec).Error)
			})
		}
	}
}
