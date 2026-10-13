package unit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/push"
)

// fakePushSubs is an in-memory PushSubscriptionRepository that records calls.
type fakePushSubs struct {
	byUser    map[string][]push.Subscription
	upsertErr error
	deleteErr error
	listErr   error

	upserted   push.Subscription
	upsertMax  int
	deletedFor [2]string
	deleted    []string
}

func (f *fakePushSubs) Upsert(_ context.Context, s push.Subscription, max int) error {
	f.upserted, f.upsertMax = s, max
	return f.upsertErr
}

func (f *fakePushSubs) DeleteForUser(_ context.Context, userID, endpoint string) error {
	f.deletedFor = [2]string{userID, endpoint}
	return f.deleteErr
}

func (f *fakePushSubs) DeleteByEndpoint(_ context.Context, endpoint string) error {
	f.deleted = append(f.deleted, endpoint)
	return f.deleteErr
}

func (f *fakePushSubs) ListByUser(_ context.Context, userID string) ([]push.Subscription, error) {
	return f.byUser[userID], f.listErr
}

func newPushService(subs *fakePushSubs) *service.PushService {
	return service.NewPushService(subs, fakeIDs{id: "sub-id"}, fakeClock{now: pushAt}, "public-key")
}

func TestPushServicePublicKey(t *testing.T) {
	assert.Equal(t, "public-key", newPushService(&fakePushSubs{}).PublicKey())
}

func TestPushServiceSubscribe(t *testing.T) {
	subs := &fakePushSubs{}

	err := newPushService(subs).Subscribe(context.Background(), "lia", "https://push.example.com/a", validP256dh, validAuth)

	require.NoError(t, err)
	assert.Equal(t, push.MaxPerUser, subs.upsertMax)
	assert.Equal(t, "sub-id", subs.upserted.ID)
	assert.Equal(t, "lia", subs.upserted.UserID)
	assert.Equal(t, "https://push.example.com/a", subs.upserted.Endpoint)
	assert.Equal(t, pushAt.Truncate(1000), subs.upserted.UpdatedAt)
}

func TestPushServiceSubscribeInvalid(t *testing.T) {
	subs := &fakePushSubs{}

	err := newPushService(subs).Subscribe(context.Background(), "lia", "http://push.example.com/a", validP256dh, validAuth)

	assert.ErrorIs(t, err, push.ErrInvalidSubscription)
	assert.Empty(t, subs.upserted.ID, "nothing is stored")
}

func TestPushServiceSubscribeRepoError(t *testing.T) {
	err := newPushService(&fakePushSubs{upsertErr: errBoom}).Subscribe(context.Background(), "lia", "https://push.example.com/a", validP256dh, validAuth)

	assert.ErrorIs(t, err, errBoom)
}

func TestPushServiceUnsubscribe(t *testing.T) {
	subs := &fakePushSubs{}

	require.NoError(t, newPushService(subs).Unsubscribe(context.Background(), "lia", "https://push.example.com/a"))
	assert.Equal(t, [2]string{"lia", "https://push.example.com/a"}, subs.deletedFor)

	subs.deleteErr = errBoom
	assert.ErrorIs(t, newPushService(subs).Unsubscribe(context.Background(), "lia", "x"), errBoom)
}
