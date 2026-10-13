// Package webpush delivers Web Push notifications (RFC 8030/8291/8292) with VAPID.
package webpush

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	wp "github.com/SherClockHolmes/webpush-go"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/push"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/reminder"
)

// Sender sends reminder payloads to browser push services.
type Sender struct {
	publicKey, privateKey, subscriber string
	client                            *http.Client
}

// NewSender builds a Sender. subject is the VAPID contact, either "mailto:you@example.com"
// or an https URL; it ends up verbatim in the signed JWT's "sub" claim.
func NewSender(publicKey, privateKey, subject string, client *http.Client) *Sender {
	// webpush-go prepends "mailto:" to any subscriber that isn't an https URL, so hand
	// it the bare address: a doubled prefix makes Apple reject the JWT (BadJwtToken).
	return &Sender{
		publicKey:  publicKey,
		privateKey: privateKey,
		subscriber: strings.TrimPrefix(subject, "mailto:"),
		client:     client,
	}
}

// Send encrypts p for s and posts it to s.Endpoint. It returns push.ErrGone when the
// push service answers 404 or 410. Errors never include the full endpoint URL.
func (s *Sender) Send(ctx context.Context, sub push.Subscription, p reminder.Payload) error {
	body, _ := json.Marshal(p) // a struct of strings always marshals
	res, err := wp.SendNotificationWithContext(ctx, body,
		&wp.Subscription{Endpoint: sub.Endpoint, Keys: wp.Keys{P256dh: sub.P256dh, Auth: sub.Auth}},
		&wp.Options{
			HTTPClient:      s.client,
			Subscriber:      s.subscriber,
			VAPIDPublicKey:  s.publicKey,
			VAPIDPrivateKey: s.privateKey,
			TTL:             reminder.TTLSeconds,
			Urgency:         wp.UrgencyNormal,
			Topic:           reminder.Topic,
		})
	if err != nil {
		// A transport error (*url.Error) quotes the endpoint; keep only its cause.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return fmt.Errorf("push: request to %s: %w", sub.Host(), urlErr.Err)
		}
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	switch {
	case res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusGone:
		return push.ErrGone
	case res.StatusCode < 200 || res.StatusCode > 299:
		return fmt.Errorf("push: status %d", res.StatusCode)
	}
	return nil
}
