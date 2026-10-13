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

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/push"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/middleware"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

type fakePushUseCase struct {
	err  error
	args []string
}

func (f *fakePushUseCase) PublicKey() string { return "the-public-key" }

func (f *fakePushUseCase) Subscribe(_ context.Context, userID, endpoint, p256dh, auth string) error {
	f.args = []string{userID, endpoint, p256dh, auth}
	return f.err
}

func (f *fakePushUseCase) Unsubscribe(_ context.Context, userID, endpoint string) error {
	f.args = []string{userID, endpoint}
	return f.err
}

func servePush(t *testing.T, uc controllers.PushUseCase, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := controllers.NewPushController(uc)
	r := gin.New()
	r.GET("/push/public-key", h.PublicKey)
	me := r.Group("/me", middleware.RequireAuth(fakeVerifier{userID: "lia"}))
	me.PUT("/push-subscription", h.Subscribe)
	me.DELETE("/push-subscription", h.Unsubscribe)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer good")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

const validPushBody = `{"endpoint":"https://push.example.com/a","expirationTime":null,"keys":{"p256dh":"P","auth":"A"}}`

func TestPushControllerPublicKey(t *testing.T) {
	rec := servePush(t, &fakePushUseCase{}, http.MethodGet, "/push/public-key", "")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, controllers.PublicKeyResponse{PublicKey: "the-public-key"}, decode[controllers.PublicKeyResponse](t, rec))
}

func TestPushControllerSubscribe(t *testing.T) {
	uc := &fakePushUseCase{}

	rec := servePush(t, uc, http.MethodPut, "/me/push-subscription", validPushBody)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, []string{"lia", "https://push.example.com/a", "P", "A"}, uc.args)
}

func TestPushControllerSubscribeErrors(t *testing.T) {
	cases := map[string]struct {
		body    string
		err     error
		status  int
		message string
	}{
		"malformed json": {`nope`, nil, http.StatusBadRequest, "invalid subscription"},
		"invalid":        {validPushBody, push.ErrInvalidSubscription, http.StatusBadRequest, "invalid subscription"},
		"unexpected":     {validPushBody, errBoom, http.StatusInternalServerError, "internal error"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rec := servePush(t, &fakePushUseCase{err: tc.err}, http.MethodPut, "/me/push-subscription", tc.body)

			require.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.message, decode[response.ErrorBody](t, rec).Error)
		})
	}
}

func TestPushControllerUnsubscribe(t *testing.T) {
	uc := &fakePushUseCase{}

	rec := servePush(t, uc, http.MethodDelete, "/me/push-subscription", `{"endpoint":"https://push.example.com/a"}`)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, []string{"lia", "https://push.example.com/a"}, uc.args)
}

func TestPushControllerUnsubscribeErrors(t *testing.T) {
	for _, body := range []string{`nope`, `{}`, `{"endpoint":""}`} {
		rec := servePush(t, &fakePushUseCase{}, http.MethodDelete, "/me/push-subscription", body)
		require.Equal(t, http.StatusBadRequest, rec.Code, body)
		assert.Equal(t, "endpoint is required", decode[response.ErrorBody](t, rec).Error)
	}

	rec := servePush(t, &fakePushUseCase{err: errBoom}, http.MethodDelete, "/me/push-subscription", `{"endpoint":"x"}`)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "internal error", decode[response.ErrorBody](t, rec).Error)
}
