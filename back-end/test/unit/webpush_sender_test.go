package unit_test

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	wp "github.com/SherClockHolmes/webpush-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/push"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/reminder"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/webpush"
)

// pushEndpoint is a fake push service that answers status and records the last request's headers.
func pushEndpoint(t *testing.T, status int, got *http.Header) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*got = r.Header.Clone()
		w.WriteHeader(status)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// browserSubscription builds a subscription to endpoint with real P-256/auth keys,
// like the ones a browser hands out.
func browserSubscription(t *testing.T, endpoint string) push.Subscription {
	t.Helper()
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	auth := make([]byte, 16)
	_, err = rand.Read(auth)
	require.NoError(t, err)
	return push.Subscription{
		UserID:   "lia",
		Endpoint: endpoint,
		P256dh:   base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()),
		Auth:     base64.RawURLEncoding.EncodeToString(auth),
	}
}

func newTestSender(t *testing.T, subject string) *webpush.Sender {
	t.Helper()
	private, public, err := wp.GenerateVAPIDKeys()
	require.NoError(t, err)
	return webpush.NewSender(public, private, subject, &http.Client{Timeout: 5 * time.Second})
}

// vapidClaims decodes the JWT in an "Authorization: vapid t=<jwt>, k=<key>" header.
func vapidClaims(t *testing.T, header string) map[string]any {
	t.Helper()
	token, ok := strings.CutPrefix(header, "vapid t=")
	require.True(t, ok, header)
	token, _, _ = strings.Cut(token, ",")
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(raw, &claims))
	return claims
}

func TestWebPushSenderSends(t *testing.T) {
	var got http.Header
	srv := pushEndpoint(t, http.StatusCreated, &got)

	err := newTestSender(t, "mailto:gabriel@example.com").Send(context.Background(), browserSubscription(t, srv.URL+"/push/abc"), reminder.Daily())

	require.NoError(t, err)
	assert.Equal(t, "10800", got.Get("TTL"))
	assert.Equal(t, "normal", got.Get("Urgency"))
	assert.Equal(t, "daily-reminder", got.Get("Topic"))
	assert.Equal(t, "aes128gcm", got.Get("Content-Encoding"))
	claims := vapidClaims(t, got.Get("Authorization"))
	assert.Equal(t, "mailto:gabriel@example.com", claims["sub"], "no doubled mailto: prefix")
	assert.Equal(t, srv.URL, claims["aud"])
}

func TestWebPushSenderKeepsHTTPSSubject(t *testing.T) {
	var got http.Header
	srv := pushEndpoint(t, http.StatusCreated, &got)

	err := newTestSender(t, "https://boraquest.example.com").Send(context.Background(), browserSubscription(t, srv.URL), reminder.Daily())

	require.NoError(t, err)
	assert.Equal(t, "https://boraquest.example.com", vapidClaims(t, got.Get("Authorization"))["sub"])
}

func TestWebPushSenderStatuses(t *testing.T) {
	cases := map[int]error{
		http.StatusNotFound: push.ErrGone,
		http.StatusGone:     push.ErrGone,
	}
	for status, want := range cases {
		var got http.Header
		srv := pushEndpoint(t, status, &got)
		err := newTestSender(t, "mailto:a@example.com").Send(context.Background(), browserSubscription(t, srv.URL), reminder.Daily())
		assert.ErrorIs(t, err, want, "status %d", status)
	}

	for _, status := range []int{http.StatusInternalServerError, http.StatusForbidden, http.StatusMovedPermanently} {
		var got http.Header
		srv := pushEndpoint(t, status, &got)
		err := newTestSender(t, "mailto:a@example.com").Send(context.Background(), browserSubscription(t, srv.URL), reminder.Daily())
		require.Error(t, err)
		assert.NotErrorIs(t, err, push.ErrGone)
		assert.Contains(t, err.Error(), "status")
	}
}

func TestWebPushSenderTransportErrorHidesEndpoint(t *testing.T) {
	var got http.Header
	srv := pushEndpoint(t, http.StatusCreated, &got)
	endpoint := srv.URL + "/secret-capability-token"
	srv.Close()

	err := newTestSender(t, "mailto:a@example.com").Send(context.Background(), browserSubscription(t, endpoint), reminder.Daily())

	require.Error(t, err)
	assert.NotErrorIs(t, err, push.ErrGone)
	assert.NotContains(t, err.Error(), "secret-capability-token")
	assert.Contains(t, err.Error(), strings.TrimPrefix(srv.URL, "http://"))
}

func TestWebPushSenderInvalidKeys(t *testing.T) {
	sub := browserSubscription(t, "https://push.example.com/x")
	sub.P256dh = base64.RawURLEncoding.EncodeToString(make([]byte, 65)) // not a point on the curve

	err := newTestSender(t, "mailto:a@example.com").Send(context.Background(), sub, reminder.Daily())

	assert.Error(t, err)
}
