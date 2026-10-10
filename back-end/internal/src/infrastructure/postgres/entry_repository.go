package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
)

const entryColumns = `id, guild_id, rule_id, rule_name, score_type, points, member_id, logged_by, occurred_on, period_key, created_at`

// EntryRepository stores entries in the entries table.
type EntryRepository struct {
	pool *pgxpool.Pool
}

// NewEntryRepository builds an EntryRepository.
func NewEntryRepository(pool *pgxpool.Pool) *EntryRepository {
	return &EntryRepository{pool: pool}
}

// Create inserts e, enforcing one sum-type entry per (rule, member, period_key)
// via an advisory lock + check-then-insert, mirroring RuleRepository.CreateWithLimit.
func (r *EntryRepository) Create(ctx context.Context, e entry.Entry) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if e.ScoreType == rule.ScoreTypeSum {
			// Serialize creations per (rule, member, period) so concurrent requests
			// can't both pass the existence check. The lock is taken in its own
			// statement so the insert below sees rows committed while waiting.
			key := e.RuleID + "|" + e.MemberID + "|" + e.PeriodKey.Format("2006-01-02")
			if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, key); err != nil {
				return err
			}
			var exists bool
			if err := tx.QueryRow(ctx,
				`SELECT EXISTS (SELECT 1 FROM entries WHERE rule_id=$1 AND member_id=$2 AND period_key=$3 AND score_type='sum')`,
				e.RuleID, e.MemberID, e.PeriodKey).Scan(&exists); err != nil {
				return err
			}
			if exists {
				return entry.ErrAlreadyLogged
			}
		}
		_, err := tx.Exec(ctx, `INSERT INTO entries (`+entryColumns+`)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			e.ID, e.GuildID, e.RuleID, e.RuleName, e.ScoreType, e.Points, e.MemberID, e.LoggedBy, e.OccurredOn, e.PeriodKey, e.CreatedAt)
		return err
	})
}

// Get returns one entry of the guild.
func (r *EntryRepository) Get(ctx context.Context, guildID, id string) (entry.Entry, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+entryColumns+` FROM entries WHERE guild_id = $1 AND id = $2`, guildID, id)
	found, err := scanEntry(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return entry.Entry{}, entry.ErrNotFound
	}
	return found, err
}

// Delete removes one entry of the guild.
func (r *EntryRepository) Delete(ctx context.Context, guildID, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM entries WHERE guild_id = $1 AND id = $2`, guildID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entry.ErrNotFound
	}
	return nil
}

// ListByMember returns memberID's entries with occurredOn in [from, to], newest first.
func (r *EntryRepository) ListByMember(ctx context.Context, guildID, memberID string, from, to time.Time) ([]entry.Entry, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+entryColumns+` FROM entries
		WHERE guild_id = $1 AND member_id = $2 AND occurred_on BETWEEN $3 AND $4
		ORDER BY occurred_on DESC, created_at DESC`,
		guildID, memberID, from, to)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (entry.Entry, error) { return scanEntry(row) })
}

// SumByGuild returns, for every member of guildID with at least one entry whose
// occurredOn falls in [from, to], their total points and count of sum-type entries.
func (r *EntryRepository) SumByGuild(ctx context.Context, guildID string, from, to time.Time) ([]service.MemberTotal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT member_id,
		       COALESCE(SUM(points), 0)::int AS points,
		       COUNT(*) FILTER (WHERE score_type = 'sum')::int AS completed
		FROM entries
		WHERE guild_id = $1 AND occurred_on BETWEEN $2 AND $3
		GROUP BY member_id`,
		guildID, from, to)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (service.MemberTotal, error) {
		var t service.MemberTotal
		err := row.Scan(&t.MemberID, &t.Points, &t.Completed)
		return t, err
	})
}

func scanEntry(row pgx.Row) (entry.Entry, error) {
	var e entry.Entry
	err := row.Scan(&e.ID, &e.GuildID, &e.RuleID, &e.RuleName, &e.ScoreType, &e.Points, &e.MemberID, &e.LoggedBy, &e.OccurredOn, &e.PeriodKey, &e.CreatedAt)
	e.OccurredOn, e.PeriodKey, e.CreatedAt = e.OccurredOn.UTC(), e.PeriodKey.UTC(), e.CreatedAt.UTC()
	return e, err
}
