package unit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

type fakeAuthenticator struct {
	token           string
	user            user.User
	err             error
	email, password string
}

func (f *fakeAuthenticator) Login(_ context.Context, email, password string) (string, user.User, error) {
	f.email, f.password = email, password
	return f.token, f.user, f.err
}

func serveLogin(t *testing.T, auth controllers.Authenticator, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controllers.NewAuthController(auth).Login(c)
	return rec
}

func TestAuthControllerLogin(t *testing.T) {
	auth := &fakeAuthenticator{token: "tok", user: lia}

	rec := serveLogin(t, auth, `{"email":"lia@boraquest.dev","password":"secret"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, controllers.LoginResponse{
		Token: "tok",
		User:  controllers.UserResponse{ID: "lia", Name: "Lia", Email: "lia@boraquest.dev"},
	}, decode[controllers.LoginResponse](t, rec))
	assert.Equal(t, "lia@boraquest.dev", auth.email)
	assert.Equal(t, "secret", auth.password)
	assert.NotContains(t, rec.Body.String(), "hash", "password hash must never leak")
}

func TestAuthControllerLoginInvalidBody(t *testing.T) {
	for _, body := range []string{`nope`, `{"email":"lia@boraquest.dev"}`, `{"password":"x"}`} {
		t.Run(body, func(t *testing.T) {
			rec := serveLogin(t, &fakeAuthenticator{}, body)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "email and password are required", decode[response.ErrorBody](t, rec).Error)
		})
	}
}

func TestAuthControllerLoginErrors(t *testing.T) {
	cases := map[string]struct {
		err     error
		status  int
		message string
	}{
		"invalid credentials": {user.ErrInvalidCredentials, http.StatusUnauthorized, "invalid email or password"},
		"unexpected":          {errBoom, http.StatusInternalServerError, "internal error"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rec := serveLogin(t, &fakeAuthenticator{err: tc.err}, `{"email":"a@b.c","password":"x"}`)

			require.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.message, decode[response.ErrorBody](t, rec).Error)
		})
	}
}
