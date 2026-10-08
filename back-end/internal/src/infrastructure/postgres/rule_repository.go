package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
)

const ruleColumns = `id, guild_id, name, frequency, score_type, score, created_by, created_at, updated_at`

// RuleRepository stores rules in the rules table.
type RuleRepository struct {
	pool *pgxpool.Pool
}

// NewRuleRepository builds a RuleRepository.
func NewRuleRepository(pool *pgxpool.Pool) *RuleRepository {
	return &RuleRepository{pool: pool}
}

// ListByGuild returns the guild's rules, oldest first.
func (r *RuleRepository) ListByGuild(ctx context.Context, guildID string) ([]rule.Rule, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+ruleColumns+` FROM rules WHERE guild_id = $1 ORDER BY created_at, id`, guildID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (rule.Rule, error) { return scanRule(row) })
}

// Get returns one rule of the guild.
func (r *RuleRepository) Get(ctx context.Context, guildID, id string) (rule.Rule, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+ruleColumns+` FROM rules WHERE guild_id = $1 AND id = $2`, guildID, id)
	found, err := scanRule(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return rule.Rule{}, rule.ErrNotFound
	}
	return found, err
}

// CreateWithLimit inserts x unless its guild already holds max rules.
func (r *RuleRepository) CreateWithLimit(ctx context.Context, x rule.Rule, max int) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		// Serialize creations per guild so concurrent requests can't both pass the count.
		// The lock is taken in its own statement so the insert below sees rows committed while waiting.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, x.GuildID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO rules (`+ruleColumns+`)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9
			WHERE (SELECT count(*) FROM rules WHERE guild_id = $2) < $10`,
			x.ID, x.GuildID, x.Name, x.Frequency, x.ScoreType, x.Score, x.CreatedBy, x.CreatedAt, x.UpdatedAt, max)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return rule.ErrLimitReached
		}
		return nil
	})
}

// Update saves the editable fields of x.
func (r *RuleRepository) Update(ctx context.Context, x rule.Rule) error {
	return expectOne(r.pool.Exec(ctx, `
		UPDATE rules SET name = $3, frequency = $4, score_type = $5, score = $6, updated_at = $7
		WHERE guild_id = $1 AND id = $2`,
		x.GuildID, x.ID, x.Name, x.Frequency, x.ScoreType, x.Score, x.UpdatedAt))
}

// Delete removes one rule of the guild.
func (r *RuleRepository) Delete(ctx context.Context, guildID, id string) error {
	return expectOne(r.pool.Exec(ctx, `DELETE FROM rules WHERE guild_id = $1 AND id = $2`, guildID, id))
}

func scanRule(row pgx.Row) (rule.Rule, error) {
	var x rule.Rule
	err := row.Scan(&x.ID, &x.GuildID, &x.Name, &x.Frequency, &x.ScoreType, &x.Score, &x.CreatedBy, &x.CreatedAt, &x.UpdatedAt)
	x.CreatedAt, x.UpdatedAt = x.CreatedAt.UTC(), x.UpdatedAt.UTC()
	return x, err
}

// expectOne maps a statement that touched no row to rule.ErrNotFound.
func expectOne(tag pgconn.CommandTag, err error) error {
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return rule.ErrNotFound
	}
	return nil
}
