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
func (r *PrizeRepository) Upsert(ctx context.Context, guildID string, period prize.Period, text string, now time.Time) (prize.Prizes, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO prizes (guild_id, week, month, updated_at)
		VALUES ($1, CASE WHEN $2 = 'week' THEN $3 ELSE '' END, CASE WHEN $2 = 'month' THEN $3 ELSE '' END, $4)
		ON CONFLICT (guild_id) DO UPDATE SET
			week       = CASE WHEN $2 = 'week'  THEN $3 ELSE prizes.week  END,
			month      = CASE WHEN $2 = 'month' THEN $3 ELSE prizes.month END,
			updated_at = $4
		RETURNING guild_id, week, month, updated_at`,
		guildID, period, text, now)
	return scanPrizes(row)
}

func scanPrizes(row pgx.Row) (prize.Prizes, error) {
	var x prize.Prizes
	err := row.Scan(&x.GuildID, &x.Week, &x.Month, &x.UpdatedAt)
	x.UpdatedAt = x.UpdatedAt.UTC()
	return x, err
}
