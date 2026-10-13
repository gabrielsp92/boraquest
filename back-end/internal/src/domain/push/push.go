// Package push holds the domain model for Web Push subscriptions.
package push

import (
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrInvalidSubscription is returned when a subscription's endpoint or keys are malformed.
	ErrInvalidSubscription = errors.New("push: invalid subscription")
	// ErrGone is returned when the push service answers 404/410: the subscription no longer exists.
	ErrGone = errors.New("push: subscription no longer valid")
)

const (
	// MaxEndpointLen caps the length of a subscription endpoint URL.
	MaxEndpointLen = 2048
	// MaxPerUser caps how many subscriptions (devices) one user may hold.
	MaxPerUser = 10

	p256dhLen = 65 // uncompressed P-256 point
	authLen   = 16
)

// Subscription is one device's Web Push endpoint for one user.
type Subscription struct {
	ID        string
	UserID    string
	Endpoint  string // https URL from PushSubscription.endpoint; unique across all users
	P256dh    string // base64url, decodes to 65 bytes
	Auth      string // base64url, decodes to 16 bytes
	CreatedAt time.Time
	UpdatedAt time.Time
}

// New validates and builds a Subscription. endpoint must parse as an absolute https URL
// of at most MaxEndpointLen bytes; p256dh and auth must be unpadded or padded base64url
// decoding to 65 and 16 bytes. Any failure returns ErrInvalidSubscription.
func New(id, userID, endpoint, p256dh, auth string, now time.Time) (Subscription, error) {
	if len(endpoint) > MaxEndpointLen {
		return Subscription{}, ErrInvalidSubscription
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return Subscription{}, ErrInvalidSubscription
	}
	if decodedLen(p256dh) != p256dhLen || decodedLen(auth) != authLen {
		return Subscription{}, ErrInvalidSubscription
	}
	now = now.Truncate(time.Microsecond)
	return Subscription{
		ID: id, UserID: userID, Endpoint: endpoint, P256dh: p256dh, Auth: auth,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// Host returns the endpoint's host, the only part of it that is safe to log
// (the full endpoint URL is a capability URL).
func (s Subscription) Host() string {
	u, err := url.Parse(s.Endpoint)
	if err != nil {
		return ""
	}
	return u.Host
}

// decodedLen returns how many bytes the base64url string s decodes to, or -1 if it is invalid.
func decodedLen(s string) int {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
	if err != nil {
		return -1
	}
	return len(b)
}
