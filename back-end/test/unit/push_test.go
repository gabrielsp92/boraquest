package unit_test

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/push"
)

// b64 returns n zero-ish bytes encoded as unpadded base64url.
func b64(n int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("k", n)))
}

var (
	validP256dh = b64(65)
	validAuth   = b64(16)
	pushAt      = time.Date(2026, 10, 13, 23, 30, 0, 123456789, time.UTC)
)

func TestPushNew(t *testing.T) {
	s, err := push.New("id1", "lia", "https://push.example.com/abc", validP256dh, validAuth, pushAt)

	require.NoError(t, err)
	want := pushAt.Truncate(time.Microsecond)
	assert.Equal(t, push.Subscription{
		ID: "id1", UserID: "lia", Endpoint: "https://push.example.com/abc", P256dh: validP256dh, Auth: validAuth,
		CreatedAt: want, UpdatedAt: want,
	}, s)
}

func TestPushNewAcceptsPaddedKeys(t *testing.T) {
	padded := base64.URLEncoding.EncodeToString([]byte(strings.Repeat("k", 16)))
	require.True(t, strings.HasSuffix(padded, "="))

	_, err := push.New("id1", "lia", "https://push.example.com/abc", validP256dh, padded, pushAt)

	assert.NoError(t, err)
}

func TestPushNewRejectsInvalid(t *testing.T) {
	cases := map[string]struct{ endpoint, p256dh, auth string }{
		"http endpoint":     {"http://push.example.com/abc", validP256dh, validAuth},
		"relative endpoint": {"/push/abc", validP256dh, validAuth},
		"no host":           {"https:///abc", validP256dh, validAuth},
		"unparseable":       {"https://push.example.com/%zz", validP256dh, validAuth},
		"too long":          {"https://push.example.com/" + strings.Repeat("a", push.MaxEndpointLen), validP256dh, validAuth},
		"bad base64":        {"https://push.example.com/abc", "not base64!", validAuth},
		"p256dh 64 bytes":   {"https://push.example.com/abc", b64(64), validAuth},
		"p256dh 66 bytes":   {"https://push.example.com/abc", b64(66), validAuth},
		"auth 15 bytes":     {"https://push.example.com/abc", validP256dh, b64(15)},
		"auth 17 bytes":     {"https://push.example.com/abc", validP256dh, b64(17)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := push.New("id1", "lia", tc.endpoint, tc.p256dh, tc.auth, pushAt)
			assert.ErrorIs(t, err, push.ErrInvalidSubscription)
		})
	}
}

func TestPushSubscriptionHost(t *testing.T) {
	assert.Equal(t, "web.push.apple.com", push.Subscription{Endpoint: "https://web.push.apple.com/secret-token"}.Host())
	assert.Equal(t, "", push.Subscription{Endpoint: "https://x/%zz"}.Host())
}
