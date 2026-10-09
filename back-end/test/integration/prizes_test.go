package integration_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

func TestPrizesFreshGuildIsEmpty(t *testing.T) {
	resetPrizes(t)
	srv := newServer(t)
	token := login(t, srv.URL, "lia@boraquest.dev")

	res := call(t, http.MethodGet, srv.URL+"/api/v1/prizes", token, nil)

	require.Equal(t, http.StatusOK, res.StatusCode)
	got := decodeBody[controllers.PrizeResponse](t, res)
	assert.Equal(t, controllers.PrizeResponse{Week: "", Month: "", UpdatedAt: time.Time{}}, got)
}

func TestPrizesSetWeekAndMonthIndependently(t *testing.T) {
	resetPrizes(t)
	srv := newServer(t)
	token := login(t, srv.URL, "lia@boraquest.dev")

	res := call(t, http.MethodPut, srv.URL+"/api/v1/prizes/week", token, controllers.SetPrizeRequest{Text: "Escolher o filme de sábado"})
	require.Equal(t, http.StatusOK, res.StatusCode)
	afterWeek := decodeBody[controllers.PrizeResponse](t, res)
	assert.Equal(t, "Escolher o filme de sábado", afterWeek.Week)
	assert.Equal(t, "", afterWeek.Month)

	res = call(t, http.MethodPut, srv.URL+"/api/v1/prizes/month", token, controllers.SetPrizeRequest{Text: "Jantar no restaurante favorito"})
	require.Equal(t, http.StatusOK, res.StatusCode)
	afterMonth := decodeBody[controllers.PrizeResponse](t, res)
	assert.Equal(t, "Escolher o filme de sábado", afterMonth.Week, "setting month must not touch week")
	assert.Equal(t, "Jantar no restaurante favorito", afterMonth.Month)

	res = call(t, http.MethodGet, srv.URL+"/api/v1/prizes", token, nil)
	require.Equal(t, http.StatusOK, res.StatusCode)
	got := decodeBody[controllers.PrizeResponse](t, res)
	assert.Equal(t, "Escolher o filme de sábado", got.Week)
	assert.Equal(t, "Jantar no restaurante favorito", got.Month)
}

func TestPrizesTooLong(t *testing.T) {
	resetPrizes(t)
	srv := newServer(t)
	token := login(t, srv.URL, "lia@boraquest.dev")
	text := strings.Repeat("a", 61)

	for _, path := range []string{"/api/v1/prizes/week", "/api/v1/prizes/month"} {
		res := call(t, http.MethodPut, srv.URL+path, token, controllers.SetPrizeRequest{Text: text})
		assert.Equal(t, http.StatusBadRequest, res.StatusCode, path)
	}
}

func TestPrizesForbidUsersOutsideTheGuild(t *testing.T) {
	resetPrizes(t)
	srv := newServer(t)
	token := login(t, srv.URL, outsiderID+"@boraquest.dev")

	res := call(t, http.MethodGet, srv.URL+"/api/v1/prizes", token, nil)
	require.Equal(t, http.StatusForbidden, res.StatusCode)
	assert.Equal(t, "you are not a member of any guild", decodeBody[response.ErrorBody](t, res).Error)

	res = call(t, http.MethodPut, srv.URL+"/api/v1/prizes/week", token, controllers.SetPrizeRequest{Text: "Cinema"})
	assert.Equal(t, http.StatusForbidden, res.StatusCode)

	res = call(t, http.MethodPut, srv.URL+"/api/v1/prizes/month", token, controllers.SetPrizeRequest{Text: "Cinema"})
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
}
