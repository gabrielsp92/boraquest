package service

import (
	"context"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/push"
)

// PushSubscriptionRepository persists Web Push subscriptions, keyed by endpoint.
type PushSubscriptionRepository interface {
	// Upsert inserts s or, when s.Endpoint exists, reassigns it to s.UserID with new keys and
	// UpdatedAt. Then, if the user has more than max rows, deletes their oldest by updated_at.
	Upsert(ctx context.Context, s push.Subscription, max int) error
	// DeleteForUser deletes the row with endpoint owned by userID; no error if absent.
	DeleteForUser(ctx context.Context, userID, endpoint string) error
	// DeleteByEndpoint deletes the row with endpoint regardless of owner; no error if absent.
	DeleteByEndpoint(ctx context.Context, endpoint string) error
	// ListByUser returns every subscription of userID.
	ListByUser(ctx context.Context, userID string) ([]push.Subscription, error)
}

// PushService manages the caller's Web Push subscriptions (one per device).
type PushService struct {
	subs      PushSubscriptionRepository
	ids       IDGenerator
	clock     Clock
	publicKey string
}

// NewPushService builds a PushService. publicKey is the VAPID public key browsers subscribe with.
func NewPushService(subs PushSubscriptionRepository, ids IDGenerator, clock Clock, publicKey string) *PushService {
	return &PushService{subs: subs, ids: ids, clock: clock, publicKey: publicKey}
}

// PublicKey returns the VAPID public key (base64url).
func (s *PushService) PublicKey() string {
	return s.publicKey
}

// Subscribe saves this device's subscription for userID, taking it over if another
// user owned the endpoint, and keeps at most push.MaxPerUser per user.
func (s *PushService) Subscribe(ctx context.Context, userID, endpoint, p256dh, auth string) error {
	sub, err := push.New(s.ids.NewID(), userID, endpoint, p256dh, auth, s.clock.Now())
	if err != nil {
		return err
	}
	return s.subs.Upsert(ctx, sub, push.MaxPerUser)
}

// Unsubscribe removes userID's subscription with that endpoint, if any.
func (s *PushService) Unsubscribe(ctx context.Context, userID, endpoint string) error {
	return s.subs.DeleteForUser(ctx, userID, endpoint)
}
