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

func TestUpdateAvatar(t *testing.T) {
	srv := newServer(t)
	token := login(t, srv.URL, "lia@boraquest.dev")
	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(), `UPDATE users SET avatar_key = 'lia' WHERE id = 'lia'`)
		require.NoError(t, err)
	})

	res := call(t, http.MethodPut, srv.URL+"/api/v1/me/avatar", token, controllers.UpdateAvatarRequest{AvatarKey: "duda"})

	require.Equal(t, http.StatusOK, res.StatusCode)
	body := decodeBody[controllers.UserResponse](t, res)
	assert.Equal(t, "duda", body.AvatarKey)

	// The change is reflected on the next login.
	res = call(t, http.MethodPost, srv.URL+"/api/v1/auth/login", "", controllers.LoginRequest{Email: "lia@boraquest.dev", Password: devPassword})
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "duda", decodeBody[controllers.LoginResponse](t, res).User.AvatarKey)

	// Saving the same avatar again is a no-op success.
	res = call(t, http.MethodPut, srv.URL+"/api/v1/me/avatar", token, controllers.UpdateAvatarRequest{AvatarKey: "duda"})
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "duda", decodeBody[controllers.UserResponse](t, res).AvatarKey)
}

func TestUpdateAvatarRejectsUnknownKey(t *testing.T) {
	srv := newServer(t)
	token := login(t, srv.URL, "lia@boraquest.dev")

	res := call(t, http.MethodPut, srv.URL+"/api/v1/me/avatar", token, controllers.UpdateAvatarRequest{AvatarKey: "astolfo"})

	require.Equal(t, http.StatusBadRequest, res.StatusCode)
	assert.Equal(t, "user: invalid avatar", decodeBody[response.ErrorBody](t, res).Error)
}

func TestUpdateAvatarRequiresAuth(t *testing.T) {
	srv := newServer(t)

	res := call(t, http.MethodPut, srv.URL+"/api/v1/me/avatar", "", controllers.UpdateAvatarRequest{AvatarKey: "duda"})

	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}
