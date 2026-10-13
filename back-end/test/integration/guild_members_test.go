package integration_test

import (
	"context"
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
		{ID: "lia", Name: "Lia", AvatarKey: "lia"},
		{ID: "beto", Name: "Beto", AvatarKey: "beto"},
		{ID: "nena", Name: "Vó Nena", AvatarKey: "nena"},
		{ID: "caio", Name: "Caio", AvatarKey: "caio"},
	}, got.Members)
}

func TestGuildMembersReflectAnotherMembersAvatarChange(t *testing.T) {
	srv := newServer(t)
	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(), `UPDATE users SET avatar_key = 'beto' WHERE id = 'beto'`)
		require.NoError(t, err)
	})
	betoToken := login(t, srv.URL, "beto@boraquest.dev")
	res := call(t, http.MethodPut, srv.URL+"/api/v1/me/avatar", betoToken, controllers.UpdateAvatarRequest{AvatarKey: "duda"})
	require.Equal(t, http.StatusOK, res.StatusCode)

	res = call(t, http.MethodGet, srv.URL+"/api/v1/guild/members", login(t, srv.URL, "lia@boraquest.dev"), nil)

	require.Equal(t, http.StatusOK, res.StatusCode)
	avatars := map[string]string{}
	for _, m := range decodeBody[controllers.GuildMemberListResponse](t, res).Members {
		avatars[m.ID] = m.AvatarKey
	}
	assert.Equal(t, map[string]string{"lia": "lia", "beto": "duda", "nena": "nena", "caio": "caio"}, avatars)
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
