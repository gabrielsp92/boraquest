package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

func TestLogin(t *testing.T) {
	srv := newServer(t)

	res := call(t, http.MethodPost, srv.URL+"/api/v1/auth/login", "", controllers.LoginRequest{Email: "Nena@Boraquest.dev", Password: devPassword})

	require.Equal(t, http.StatusOK, res.StatusCode)
	body := decodeBody[controllers.LoginResponse](t, res)
	assert.NotEmpty(t, body.Token)
	assert.Equal(t, controllers.UserResponse{ID: "nena", Name: "Vó Nena", Email: "nena@boraquest.dev"}, body.User)
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	srv := newServer(t)

	for name, req := range map[string]controllers.LoginRequest{
		"wrong password": {Email: "lia@boraquest.dev", Password: "nope"},
		"unknown email":  {Email: "ghost@boraquest.dev", Password: devPassword},
	} {
		t.Run(name, func(t *testing.T) {
			res := call(t, http.MethodPost, srv.URL+"/api/v1/auth/login", "", req)

			require.Equal(t, http.StatusUnauthorized, res.StatusCode)
			assert.Equal(t, "invalid email or password", decodeBody[response.ErrorBody](t, res).Error)
		})
	}
}
