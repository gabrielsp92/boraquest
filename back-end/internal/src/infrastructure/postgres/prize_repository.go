package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/prize"
)

// PrizeRepository stores prizes in the prizes table.
type PrizeRepository struct {
	pool *pgxpool.Pool
}

// NewPrizeRepository builds a PrizeRepository.
func NewPrizeRepository(pool *pgxpool.Pool) *PrizeRepository {
	return &PrizeRepository{pool: pool}
}

// Get returns the guild's prizes, or a zero Prizes{GuildID: guildID} if no row exists yet.
func (r *PrizeRepository) Get(ctx context.Context, guildID string) (prize.Prizes, error) {
	row := r.pool.QueryRow(ctx, `SELECT guild_id, week, month, updated_at FROM prizes WHERE guild_id = $1`, guildID)
	found, err := scanPrizes(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return prize.Prizes{GuildID: guildID}, nil
	}
	return found, err
}

// Upsert creates the guild's row if missing, then sets just the given period's column.
// Every edit is also appended to prize_changes, in the same transaction, so closed
// weeks can later snapshot the prize that was in effect when they ended.
func (r *PrizeRepository) Upsert(ctx context.Context, guildID string, period prize.Period, text string, now time.Time) (prize.Prizes, error) {
	var x prize.Prizes
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			INSERT INTO prizes (guild_id, week, month, updated_at)
			VALUES ($1, CASE WHEN $2 = 'week' THEN $3 ELSE '' END, CASE WHEN $2 = 'month' THEN $3 ELSE '' END, $4)
			ON CONFLICT (guild_id) DO UPDATE SET
				week       = CASE WHEN $2 = 'week'  THEN $3 ELSE prizes.week  END,
				month      = CASE WHEN $2 = 'month' THEN $3 ELSE prizes.month END,
				updated_at = $4
			RETURNING guild_id, week, month, updated_at`,
			guildID, period, text, now)
		var err error
		if x, err = scanPrizes(row); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO prize_changes (guild_id, period, text, changed_at) VALUES ($1, $2, $3, $4)`,
			guildID, period, text, now)
		return err
	})
	return x, err
}

// WeekPrizeAt returns the guild's week prize as it was right before at, or ""
// if none had been set by then.
func (r *PrizeRepository) WeekPrizeAt(ctx context.Context, guildID string, at time.Time) (string, error) {
	var text string
	err := r.pool.QueryRow(ctx, `
		SELECT text FROM prize_changes
		WHERE guild_id = $1 AND period = 'week' AND changed_at < $2
		ORDER BY changed_at DESC, id DESC
		LIMIT 1`,
		guildID, at).Scan(&text)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return text, err
}

func scanPrizes(row pgx.Row) (prize.Prizes, error) {
	var x prize.Prizes
	err := row.Scan(&x.GuildID, &x.Week, &x.Month, &x.UpdatedAt)
	x.UpdatedAt = x.UpdatedAt.UTC()
	return x, err
}
