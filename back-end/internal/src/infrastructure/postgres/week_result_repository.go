package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/weekresult"
)

// WeekResultRepository stores closed weeks in week_results and week_result_standings.
type WeekResultRepository struct {
	pool *pgxpool.Pool
}

// NewWeekResultRepository builds a WeekResultRepository.
func NewWeekResultRepository(pool *pgxpool.Pool) *WeekResultRepository {
	return &WeekResultRepository{pool: pool}
}

// LatestWeekStart returns the guild's most recent closed week, or false if none.
func (r *WeekResultRepository) LatestWeekStart(ctx context.Context, guildID string) (time.Time, bool, error) {
	var ws time.Time
	err := r.pool.QueryRow(ctx, `SELECT week_start FROM week_results WHERE guild_id = $1 ORDER BY week_start DESC LIMIT 1`, guildID).Scan(&ws)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	return ws.UTC(), true, nil
}

// SaveAll stores results in one transaction. A week already stored (e.g. by a
// concurrent request) is skipped along with its standings.
func (r *WeekResultRepository) SaveAll(ctx context.Context, results []weekresult.WeekResult) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		for _, w := range results {
			tag, err := tx.Exec(ctx, `
				INSERT INTO week_results (guild_id, week_start, week_end, prize, closed_at)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (guild_id, week_start) DO NOTHING`,
				w.GuildID, w.WeekStart, w.WeekEnd, w.Prize, w.ClosedAt)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				continue
			}
			for _, s := range w.Standings {
				if _, err := tx.Exec(ctx, `
					INSERT INTO week_result_standings (guild_id, week_start, member_id, rank, points, completed)
					VALUES ($1, $2, $3, $4, $5, $6)`,
					w.GuildID, w.WeekStart, s.MemberID, s.Rank, s.Points, s.Completed); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// List returns every closed week of the guild, newest first, standings by rank.
func (r *WeekResultRepository) List(ctx context.Context, guildID string) ([]weekresult.WeekResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT w.guild_id, w.week_start, w.week_end, w.prize, w.closed_at,
		       s.member_id, s.rank, s.points, s.completed
		FROM week_results w
		JOIN week_result_standings s ON s.guild_id = w.guild_id AND s.week_start = w.week_start
		WHERE w.guild_id = $1
		ORDER BY w.week_start DESC, s.rank`,
		guildID)
	if err != nil {
		return nil, err
	}
	flat, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (weekStandingRow, error) {
		var j weekStandingRow
		err := row.Scan(&j.week.GuildID, &j.week.WeekStart, &j.week.WeekEnd, &j.week.Prize, &j.week.ClosedAt,
			&j.standing.MemberID, &j.standing.Rank, &j.standing.Points, &j.standing.Completed)
		return j, err
	})
	return groupWeeks(flat), err
}

// weekStandingRow is one row of List's join: a week with one of its standings.
type weekStandingRow struct {
	week     weekresult.WeekResult
	standing weekresult.Standing
}

// groupWeeks folds rows ordered by (week_start DESC, rank) into one WeekResult per week.
func groupWeeks(flat []weekStandingRow) []weekresult.WeekResult {
	results := []weekresult.WeekResult{}
	for _, j := range flat {
		w := j.week
		w.WeekStart, w.WeekEnd, w.ClosedAt = w.WeekStart.UTC(), w.WeekEnd.UTC(), w.ClosedAt.UTC()
		if n := len(results); n == 0 || !results[n-1].WeekStart.Equal(w.WeekStart) {
			results = append(results, w)
		}
		last := &results[len(results)-1]
		last.Standings = append(last.Standings, j.standing)
	}
	return results
}
