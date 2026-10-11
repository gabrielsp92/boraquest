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
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

type fakeProfileUseCase struct {
	user   user.User
	err    error
	userID string
	key    string
}

func (f *fakeProfileUseCase) UpdateAvatar(_ context.Context, userID, avatarKey string) (user.User, error) {
	f.userID, f.key = userID, avatarKey
	return f.user, f.err
}

func serveProfile(t *testing.T, uc controllers.ProfileUseCase, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := controllers.NewProfileController(uc)
	r := gin.New()
	g := r.Group("/me", middleware.RequireAuth(fakeVerifier{userID: "lia"}))
	g.PUT("/avatar", h.UpdateAvatar)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestProfileControllerUpdateAvatar(t *testing.T) {
	uc := &fakeProfileUseCase{user: user.User{ID: "lia", Name: "Lia", Email: "lia@boraquest.dev", AvatarKey: "duda"}}

	rec := serveProfile(t, uc, http.MethodPut, "/me/avatar", `{"avatarKey":"duda"}`)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, controllers.UserResponse{ID: "lia", Name: "Lia", Email: "lia@boraquest.dev", AvatarKey: "duda"}, decode[controllers.UserResponse](t, rec))
	assert.Equal(t, "lia", uc.userID)
	assert.Equal(t, "duda", uc.key)
}

func TestProfileControllerUpdateAvatarInvalidBody(t *testing.T) {
	for _, body := range []string{`nope`, `{}`, `{"avatarKey":""}`} {
		t.Run(body, func(t *testing.T) {
			rec := serveProfile(t, &fakeProfileUseCase{}, http.MethodPut, "/me/avatar", body)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "avatarKey is required", decode[response.ErrorBody](t, rec).Error)
		})
	}
}

func TestProfileControllerUpdateAvatarErrors(t *testing.T) {
	cases := map[string]struct {
		err     error
		status  int
		message string
	}{
		"invalid avatar": {user.ErrInvalidAvatar, http.StatusBadRequest, user.ErrInvalidAvatar.Error()},
		"not found":      {user.ErrNotFound, http.StatusNotFound, "user not found"},
		"unexpected":     {errBoom, http.StatusInternalServerError, "internal error"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rec := serveProfile(t, &fakeProfileUseCase{err: tc.err}, http.MethodPut, "/me/avatar", `{"avatarKey":"astolfo"}`)

			require.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.message, decode[response.ErrorBody](t, rec).Error)
		})
	}
}
