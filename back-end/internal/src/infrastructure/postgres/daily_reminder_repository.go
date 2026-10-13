package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DailyReminderRepository records sent daily reminders in the daily_reminders table.
type DailyReminderRepository struct {
	pool *pgxpool.Pool
}

// NewDailyReminderRepository builds a DailyReminderRepository.
func NewDailyReminderRepository(pool *pgxpool.Pool) *DailyReminderRepository {
	return &DailyReminderRepository{pool: pool}
}

// Claim inserts (userID, day) and reports true, or reports false if it already existed.
func (r *DailyReminderRepository) Claim(ctx context.Context, userID string, day, now time.Time) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO daily_reminders (user_id, day, sent_at) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, day) DO NOTHING`, userID, day, now)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
