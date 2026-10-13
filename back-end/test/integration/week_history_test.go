package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/prize"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/weekresult"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/postgres"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// resetWeekHistory empties closed weeks, the prize log, prizes and entries.
func resetWeekHistory(t *testing.T) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `DELETE FROM week_results; DELETE FROM prize_changes; DELETE FROM prizes; DELETE FROM entries`)
	require.NoError(t, err)
}

// seedPastEntry inserts an entry directly: the API can only log entries for today.
func seedPastEntry(t *testing.T, memberID, scoreType string, points int, on time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO entries (id, guild_id, rule_id, rule_name, score_type, points, member_id, logged_by, occurred_on, period_key, created_at)
		VALUES (gen_random_uuid()::text, 'familia', gen_random_uuid()::text, 'Quest', $1, $2, $3, $3, $4, $4, now())`,
		scoreType, points, memberID, on)
	require.NoError(t, err)
}

func seedPrizeChange(t *testing.T, text string, at time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `INSERT INTO prize_changes (guild_id, period, text, changed_at) VALUES ('familia', 'week', $1, $2)`, text, at)
	require.NoError(t, err)
}

// mondayStart is 00:00 in São Paulo on the given Monday.
func mondayStart(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, saoPaulo)
}

func TestWeekHistoryClosesEndedWeeksOnceWithPrizeSnapshot(t *testing.T) {
	resetWeekHistory(t)
	current := entry.WeekStart(entry.DateOnly(time.Now().In(saoPaulo)))
	w0, w1, w2 := current.AddDate(0, 0, -28), current.AddDate(0, 0, -14), current.AddDate(0, 0, -7) // current-21 stays empty

	seedPastEntry(t, "beto", "sum", 10, w0.AddDate(0, 0, 2))
	seedPastEntry(t, "nena", "sum", 20, w1)
	seedPastEntry(t, "nena", "sum", 20, w1.AddDate(0, 0, 6)) // Sunday still belongs to w1
	seedPastEntry(t, "lia", "decrease", -5, w1.AddDate(0, 0, 3))
	seedPastEntry(t, "caio", "sum", 15, w2.AddDate(0, 0, 1))
	seedPastEntry(t, "beto", "sum", 15, w2.AddDate(0, 0, 4))
	seedPastEntry(t, "lia", "sum", 99, current) // the current week is never closed

	seedPrizeChange(t, "Escolher o filme", mondayStart(w1).Add(-time.Hour))
	seedPrizeChange(t, "Pizza", mondayStart(w2).Add(time.Hour)) // set after w1 ended: only w2 gets it

	srv := newServer(t)
	token := login(t, srv.URL, "lia@boraquest.dev")

	// Editing the prize now logs it, but must not leak into already-ended weeks.
	res := call(t, http.MethodPut, srv.URL+"/api/v1/prizes/week", token, controllers.SetPrizeRequest{Text: "Sorvete"})
	require.Equal(t, http.StatusOK, res.StatusCode)
	var logged string
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT text FROM prize_changes WHERE guild_id = 'familia' AND period = 'week' ORDER BY changed_at DESC LIMIT 1`).Scan(&logged))
	assert.Equal(t, "Sorvete", logged)

	res = call(t, http.MethodGet, srv.URL+"/api/v1/weeks/history", token, nil)
	require.Equal(t, http.StatusOK, res.StatusCode)
	first := decodeBody[controllers.WeekHistoryResponse](t, res)

	date := func(d time.Time) string { return d.Format("2006-01-02") }
	require.Len(t, first.Weeks, 3)
	w2Week, w1Week, w0Week := first.Weeks[0], first.Weeks[1], first.Weeks[2]

	assert.Equal(t, date(w2), w2Week.From)
	assert.Equal(t, date(w2.AddDate(0, 0, 6)), w2Week.To)
	assert.Equal(t, "Pizza", w2Week.Prize)
	// beto and caio tie on points and completed: member id breaks the tie.
	assert.Equal(t, controllers.WeekStandingResponse{MemberID: "beto", Rank: 1, Points: 15, Completed: 1}, w2Week.Winner)
	assert.Equal(t, "caio", w2Week.Standings[1].MemberID)

	assert.Equal(t, date(w1), w1Week.From)
	assert.Equal(t, "Escolher o filme", w1Week.Prize)
	assert.Equal(t, controllers.WeekStandingResponse{MemberID: "nena", Rank: 1, Points: 40, Completed: 2}, w1Week.Winner)
	require.Len(t, w1Week.Standings, 4)
	assert.Equal(t, controllers.WeekStandingResponse{MemberID: "lia", Rank: 4, Points: -5, Completed: 0}, w1Week.Standings[3])

	assert.Equal(t, date(w0), w0Week.From)
	assert.Equal(t, "", w0Week.Prize, "no prize had been set yet")
	assert.Equal(t, "beto", w0Week.Winner.MemberID)

	// Results are frozen: a late row in a closed week changes nothing, and nothing is closed twice.
	seedPastEntry(t, "lia", "sum", 500, w1.AddDate(0, 0, 1))
	res = call(t, http.MethodGet, srv.URL+"/api/v1/weeks/history", token, nil)
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, first, decodeBody[controllers.WeekHistoryResponse](t, res))
	var closed int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM week_results`).Scan(&closed))
	assert.Equal(t, 3, closed)
}

func TestWeekHistoryEmptyGuild(t *testing.T) {
	resetWeekHistory(t)
	srv := newServer(t)
	token := login(t, srv.URL, "lia@boraquest.dev")

	res := call(t, http.MethodGet, srv.URL+"/api/v1/weeks/history", token, nil)

	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Empty(t, decodeBody[controllers.WeekHistoryResponse](t, res).Weeks)
}

func TestWeekHistoryRequiresAuth(t *testing.T) {
	srv := newServer(t)

	for name, token := range map[string]string{"missing": "", "invalid": "garbage"} {
		t.Run(name, func(t *testing.T) {
			res := call(t, http.MethodGet, srv.URL+"/api/v1/weeks/history", token, nil)
			assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
		})
	}
}

func TestWeekHistoryForbidsUsersOutsideTheGuild(t *testing.T) {
	srv := newServer(t)
	token := login(t, srv.URL, outsiderID+"@boraquest.dev")

	res := call(t, http.MethodGet, srv.URL+"/api/v1/weeks/history", token, nil)

	require.Equal(t, http.StatusForbidden, res.StatusCode)
	assert.Equal(t, "you are not a member of any guild", decodeBody[response.ErrorBody](t, res).Error)
}

func TestWeekResultRepositorySaveAllSkipsWeeksAlreadyStored(t *testing.T) {
	resetWeekHistory(t)
	ctx := context.Background()
	repo := postgres.NewWeekResultRepository(pool)
	ws := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	w := weekresult.WeekResult{GuildID: "familia", WeekStart: ws, WeekEnd: ws.AddDate(0, 0, 6), Prize: "Filme", ClosedAt: time.Now(),
		Standings: []weekresult.Standing{{MemberID: "lia", Rank: 1, Points: 10, Completed: 1}}}
	require.NoError(t, repo.SaveAll(ctx, []weekresult.WeekResult{w}))

	again := w
	again.Prize, again.Standings = "Outro", []weekresult.Standing{{MemberID: "beto", Rank: 1, Points: 99}}
	require.NoError(t, repo.SaveAll(ctx, []weekresult.WeekResult{again}))

	got, err := repo.List(ctx, "familia")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Filme", got[0].Prize)
	assert.Equal(t, []weekresult.Standing{{MemberID: "lia", Rank: 1, Points: 10, Completed: 1}}, got[0].Standings)
	latest, ok, err := repo.LatestWeekStart(ctx, "familia")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, ws, latest)
}

func TestWeekResultRepositorySaveAllStatementErrors(t *testing.T) {
	resetWeekHistory(t)
	ctx := context.Background()
	repo := postgres.NewWeekResultRepository(pool)
	ws := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	w := weekresult.WeekResult{GuildID: "\xff", WeekStart: ws, WeekEnd: ws.AddDate(0, 0, 6), ClosedAt: time.Now()}

	assert.Error(t, repo.SaveAll(ctx, []weekresult.WeekResult{w}), "invalid UTF-8 guild id")

	w.GuildID, w.Standings = "familia", []weekresult.Standing{{MemberID: "ghost", Rank: 1}}
	assert.Error(t, repo.SaveAll(ctx, []weekresult.WeekResult{w}), "unknown member violates the users foreign key")
	_, ok, err := repo.LatestWeekStart(ctx, "familia")
	require.NoError(t, err)
	assert.False(t, ok, "the failed save is rolled back")
}

func TestPrizeRepositoryUpsertStatementErrors(t *testing.T) {
	resetWeekHistory(t)
	ctx := context.Background()
	prizes := postgres.NewPrizeRepository(pool)

	_, err := prizes.Upsert(ctx, "familia", prize.PeriodWeek, fmt.Sprintf("%061d", 0), time.Now())
	assert.Error(t, err, "prizes length check")

	_, err = prizes.Upsert(ctx, "familia", prize.Period("year"), "x", time.Now())
	assert.Error(t, err, "prize_changes period check")
	got, err := prizes.Get(ctx, "familia")
	require.NoError(t, err)
	assert.Equal(t, prize.Prizes{GuildID: "familia"}, got, "the failed upsert is rolled back")
}

func TestWeekHistoryRepositoriesSurfaceDatabaseErrors(t *testing.T) {
	ctx := context.Background()
	closed, err := postgres.Open(ctx, databaseURL)
	require.NoError(t, err)
	closed.Close()
	now := time.Now()

	results := postgres.NewWeekResultRepository(closed)
	_, _, err = results.LatestWeekStart(ctx, "familia")
	assert.Error(t, err)
	assert.Error(t, results.SaveAll(ctx, nil))
	_, err = results.List(ctx, "familia")
	assert.Error(t, err)

	_, err = postgres.NewEntryRepository(closed).WeekStartsWithEntries(ctx, "familia", now, now)
	assert.Error(t, err)

	prizes := postgres.NewPrizeRepository(closed)
	_, err = prizes.WeekPrizeAt(ctx, "familia", now)
	assert.Error(t, err)
	_, err = prizes.Upsert(ctx, "familia", prize.PeriodWeek, "x", now)
	assert.Error(t, err)
}
