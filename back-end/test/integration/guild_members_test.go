package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

func TestGuildMembersListsSeededFamily(t *testing.T) {
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")

	res := call(t, http.MethodGet, srv.URL+"/api/v1/guild/members", liaToken, nil)

	require.Equal(t, http.StatusOK, res.StatusCode)
	got := decodeBody[controllers.GuildMemberListResponse](t, res)
	assert.ElementsMatch(t, []controllers.GuildMemberResponse{
		{ID: "lia", Name: "Lia"},
		{ID: "beto", Name: "Beto"},
		{ID: "nena", Name: "Vó Nena"},
		{ID: "caio", Name: "Caio"},
	}, got.Members)
}

func TestGuildMembersRequiresAuth(t *testing.T) {
	srv := newServer(t)

	for name, token := range map[string]string{"missing": "", "invalid": "garbage"} {
		t.Run(name, func(t *testing.T) {
			res := call(t, http.MethodGet, srv.URL+"/api/v1/guild/members", token, nil)
			assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
		})
	}
}

func TestGuildMembersForbidUsersOutsideTheGuild(t *testing.T) {
	srv := newServer(t)
	token := login(t, srv.URL, outsiderID+"@boraquest.dev")

	res := call(t, http.MethodGet, srv.URL+"/api/v1/guild/members", token, nil)

	require.Equal(t, http.StatusForbidden, res.StatusCode)
	assert.Equal(t, "you are not a member of any guild", decodeBody[response.ErrorBody](t, res).Error)
}
