package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/push"
)

// PushSubscriptionRepository stores Web Push subscriptions in the push_subscriptions table.
type PushSubscriptionRepository struct {
	pool *pgxpool.Pool
}

// NewPushSubscriptionRepository builds a PushSubscriptionRepository.
func NewPushSubscriptionRepository(pool *pgxpool.Pool) *PushSubscriptionRepository {
	return &PushSubscriptionRepository{pool: pool}
}

// Upsert inserts s or, when s.Endpoint exists, reassigns it to s.UserID with new keys and
// UpdatedAt (keeping its id and created_at). In the same statement it deletes the user's
// oldest other rows by updated_at so they keep at most max in total.
func (r *PushSubscriptionRepository) Upsert(ctx context.Context, s push.Subscription, max int) error {
	// Both parts of the statement see the same snapshot, so the DELETE ranks only the
	// user's other endpoints and keeps max-1 of them next to the one being saved.
	_, err := r.pool.Exec(ctx, `
		WITH saved AS (
			INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (endpoint) DO UPDATE SET
				user_id    = EXCLUDED.user_id,
				p256dh     = EXCLUDED.p256dh,
				auth       = EXCLUDED.auth,
				updated_at = EXCLUDED.updated_at
			RETURNING id
		)
		DELETE FROM push_subscriptions
		WHERE user_id = $2 AND endpoint <> $3 AND id NOT IN (
			SELECT id FROM push_subscriptions
			WHERE user_id = $2 AND endpoint <> $3
			ORDER BY updated_at DESC, id DESC
			LIMIT $8 - 1
		)`,
		s.ID, s.UserID, s.Endpoint, s.P256dh, s.Auth, s.CreatedAt, s.UpdatedAt, max)
	return err
}

// DeleteForUser deletes the row with endpoint owned by userID; no error if absent.
func (r *PushSubscriptionRepository) DeleteForUser(ctx context.Context, userID, endpoint string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`, userID, endpoint)
	return err
}

// DeleteByEndpoint deletes the row with endpoint regardless of owner; no error if absent.
func (r *PushSubscriptionRepository) DeleteByEndpoint(ctx context.Context, endpoint string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE endpoint = $1`, endpoint)
	return err
}

// ListByUser returns every subscription of userID, most recently updated first.
func (r *PushSubscriptionRepository) ListByUser(ctx context.Context, userID string) ([]push.Subscription, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, endpoint, p256dh, auth, created_at, updated_at
		FROM push_subscriptions WHERE user_id = $1
		ORDER BY updated_at DESC, id DESC`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (push.Subscription, error) {
		var s push.Subscription
		err := row.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth, &s.CreatedAt, &s.UpdatedAt)
		s.CreatedAt, s.UpdatedAt = s.CreatedAt.UTC(), s.UpdatedAt.UTC()
		return s, err
	})
}
