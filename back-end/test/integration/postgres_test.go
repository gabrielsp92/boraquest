package integration_test

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/entry"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/push"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/postgres"
)

func TestOpenErrors(t *testing.T) {
	ctx := context.Background()

	_, err := postgres.Open(ctx, "://not-a-url")
	assert.Error(t, err)

	_, err = postgres.Open(ctx, "postgres://u:p@127.0.0.1:1/db?sslmode=disable&connect_timeout=1")
	assert.Error(t, err)
}

// freshDatabase creates an empty database in the test container and returns its URL.
func freshDatabase(t *testing.T, name string) string {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `CREATE DATABASE `+name)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DROP DATABASE `+name+` WITH (FORCE)`) })
	return strings.Replace(databaseURL, "/boraquest?", "/"+name+"?", 1)
}

func TestMigrateFSErrors(t *testing.T) {
	ctx := context.Background()
	scratch, err := postgres.Open(ctx, freshDatabase(t, "migrate_errors"))
	require.NoError(t, err)
	defer scratch.Close()

	assert.ErrorIs(t, postgres.MigrateFS(ctx, scratch, fstest.MapFS{}), goose.ErrNoMigrations)

	bad := fstest.MapFS{"0001_bad.sql": {Data: []byte("-- +goose Up\nTHIS IS NOT SQL;\n")}}
	assert.Error(t, postgres.MigrateFS(ctx, scratch, bad))
}

func TestMigrateIsIdempotent(t *testing.T) {
	assert.NoError(t, postgres.Migrate(context.Background(), pool))
}

func TestRepositoriesSurfaceDatabaseErrors(t *testing.T) {
	ctx := context.Background()
	closed, err := postgres.Open(ctx, databaseURL)
	require.NoError(t, err)
	closed.Close()
	rules := postgres.NewRuleRepository(closed)
	r := rule.Rule{ID: "r1", GuildID: "g1"}

	_, err = rules.ListByGuild(ctx, "g1")
	assert.Error(t, err)
	_, err = rules.Get(ctx, "g1", "r1")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, rule.ErrNotFound)
	assert.Error(t, rules.CreateWithLimit(ctx, r, rule.MaxPerGuild))
	err = rules.Update(ctx, r)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, rule.ErrNotFound)
	err = rules.Delete(ctx, "g1", "r1")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, rule.ErrNotFound)

	_, err = postgres.NewUserRepository(closed).FindByEmail(ctx, "lia@boraquest.dev")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, user.ErrNotFound)
	err = postgres.NewUserRepository(closed).Create(ctx, user.User{ID: "x", Email: "x@boraquest.dev"}, "familia")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, user.ErrAlreadyExists)

	_, err = postgres.NewUserRepository(closed).ListByIDs(ctx, []string{"lia"})
	assert.Error(t, err)

	guilds := postgres.NewGuildRepository(closed)
	_, err = guilds.Get(ctx, "familia")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, guild.ErrNotFound)
	_, err = guilds.FindByMember(ctx, "lia")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, guild.ErrNotMember)

	entries := postgres.NewEntryRepository(closed)
	e := entry.Entry{ID: "e1", GuildID: "g1", RuleID: "r1", MemberID: "lia", LoggedBy: "lia", ScoreType: rule.ScoreTypeSum}
	assert.Error(t, entries.Create(ctx, e))
	_, err = entries.Get(ctx, "g1", "e1")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, entry.ErrNotFound)
	err = entries.Delete(ctx, "g1", "e1")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, entry.ErrNotFound)
	_, err = entries.ListByMember(ctx, "g1", "lia", time.Now(), time.Now())
	assert.Error(t, err)
	_, err = entries.SumByGuild(ctx, "g1", time.Now(), time.Now())
	assert.Error(t, err)
	_, err = entries.SetPhoto(ctx, "g1", "e1", "u", "p")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, entry.ErrNotFound)

	_, err = guilds.ListAll(ctx)
	assert.Error(t, err)

	subs := postgres.NewPushSubscriptionRepository(closed)
	assert.Error(t, subs.Upsert(ctx, push.Subscription{ID: "s1", UserID: "lia", Endpoint: "https://push.test/x"}, push.MaxPerUser))
	assert.Error(t, subs.DeleteForUser(ctx, "lia", "https://push.test/x"))
	assert.Error(t, subs.DeleteByEndpoint(ctx, "https://push.test/x"))
	_, err = subs.ListByUser(ctx, "lia")
	assert.Error(t, err)

	_, err = postgres.NewDailyReminderRepository(closed).Claim(ctx, "lia", time.Now(), time.Now())
	assert.Error(t, err)
}

func TestPushSubscriptionRepository(t *testing.T) {
	resetPush(t)
	ctx := context.Background()
	subs := postgres.NewPushSubscriptionRepository(pool)
	at := time.Date(2026, 10, 13, 23, 0, 0, 0, time.UTC)
	s := push.Subscription{ID: "s1", UserID: "lia", Endpoint: "https://push.test/a", P256dh: "p", Auth: "a", CreatedAt: at, UpdatedAt: at}
	require.NoError(t, subs.Upsert(ctx, s, push.MaxPerUser))

	got, err := subs.ListByUser(ctx, "lia")
	require.NoError(t, err)
	assert.Equal(t, []push.Subscription{s}, got)

	// Re-saving the endpoint keeps its id and created_at, and replaces owner, keys and updated_at.
	later := at.Add(time.Hour)
	moved := push.Subscription{ID: "s2", UserID: "beto", Endpoint: s.Endpoint, P256dh: "p2", Auth: "a2", CreatedAt: later, UpdatedAt: later}
	require.NoError(t, subs.Upsert(ctx, moved, push.MaxPerUser))
	got, err = subs.ListByUser(ctx, "beto")
	require.NoError(t, err)
	assert.Equal(t, []push.Subscription{{ID: "s1", UserID: "beto", Endpoint: s.Endpoint, P256dh: "p2", Auth: "a2", CreatedAt: at, UpdatedAt: later}}, got)
	got, err = subs.ListByUser(ctx, "lia")
	require.NoError(t, err)
	assert.Empty(t, got)

	// An unknown user violates the users foreign key.
	ghost := s
	ghost.ID, ghost.UserID, ghost.Endpoint = "s3", "ghost", "https://push.test/ghost"
	assert.Error(t, subs.Upsert(ctx, ghost, push.MaxPerUser))

	require.NoError(t, subs.DeleteByEndpoint(ctx, s.Endpoint))
	require.NoError(t, subs.DeleteByEndpoint(ctx, s.Endpoint), "absent is not an error")
	got, err = subs.ListByUser(ctx, "beto")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestDailyReminderRepositoryClaim(t *testing.T) {
	resetPush(t)
	ctx := context.Background()
	log := postgres.NewDailyReminderRepository(pool)
	day := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)

	ok, err := log.Claim(ctx, "lia", day, time.Now())
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = log.Claim(ctx, "lia", day, time.Now())
	require.NoError(t, err)
	assert.False(t, ok, "a second claim for the same day loses")

	ok, err = log.Claim(ctx, "lia", day.AddDate(0, 0, 1), time.Now())
	require.NoError(t, err)
	assert.True(t, ok, "the next day is a new claim")
}

func TestEntryRepositoryCreateStatementErrors(t *testing.T) {
	resetEntries(t)
	ctx := context.Background()
	entries := postgres.NewEntryRepository(pool)
	now := time.Now()

	// Invalid UTF-8 in the advisory-lock key (via RuleID) is rejected.
	bad := entry.Entry{
		ID: "e1", GuildID: "familia", RuleID: "\xff", RuleName: "x", ScoreType: rule.ScoreTypeSum, Points: 1,
		MemberID: "lia", LoggedBy: "lia", OccurredOn: now, PeriodKey: now, CreatedAt: now,
	}
	assert.Error(t, entries.Create(ctx, bad))

	// An unknown member violates the users foreign key on insert.
	unknownMember := bad
	unknownMember.RuleID, unknownMember.MemberID = "r1", "ghost"
	err := entries.Create(ctx, unknownMember)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, entry.ErrAlreadyLogged)

	// A period_key outside Postgres's representable date range fails the
	// exists check itself, after the advisory lock already succeeded.
	outOfRange := bad
	outOfRange.RuleID, outOfRange.PeriodKey = "r1", time.Date(6000000, 1, 1, 0, 0, 0, 0, time.UTC)
	err = entries.Create(ctx, outOfRange)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, entry.ErrAlreadyLogged)
}

func TestEntryRepositoryDeleteNotFound(t *testing.T) {
	resetEntries(t)
	entries := postgres.NewEntryRepository(pool)

	assert.ErrorIs(t, entries.Delete(context.Background(), "familia", "ghost-id"), entry.ErrNotFound)
}

func TestCreateWithLimitStatementErrors(t *testing.T) {
	resetRules(t)
	ctx := context.Background()
	rules := postgres.NewRuleRepository(pool)
	now := time.Now()

	// Invalid UTF-8 is rejected by the advisory-lock statement.
	badGuild := rule.Rule{ID: "r1", GuildID: "\xff", Name: "x", Frequency: rule.FrequencyDaily, ScoreType: rule.ScoreTypeSum, Score: 1, CreatedBy: "lia", CreatedAt: now, UpdatedAt: now}
	assert.Error(t, rules.CreateWithLimit(ctx, badGuild, rule.MaxPerGuild))

	// An unknown creator violates the users foreign key on insert.
	unknownCreator := badGuild
	unknownCreator.GuildID, unknownCreator.CreatedBy = "familia", "ghost"
	err := rules.CreateWithLimit(ctx, unknownCreator, rule.MaxPerGuild)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, rule.ErrLimitReached)
}

func TestUserRepositoryFindByEmail(t *testing.T) {
	users := postgres.NewUserRepository(pool)

	u, err := users.FindByEmail(context.Background(), "beto@boraquest.dev")
	require.NoError(t, err)
	assert.Equal(t, "beto", u.ID)
	assert.True(t, strings.HasPrefix(u.PasswordHash, "$2a$"), "stores a bcrypt hash")

	_, err = users.FindByEmail(context.Background(), "ghost@boraquest.dev")
	assert.ErrorIs(t, err, user.ErrNotFound)
}

func TestUserRepositoryCreate(t *testing.T) {
	ctx := context.Background()
	users := postgres.NewUserRepository(pool)
	ana := user.User{ID: "ana-create", Name: "Ana", Email: "ana-create@boraquest.dev", PasswordHash: "hash", AvatarKey: "lia"}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM users WHERE id LIKE 'ana-create%'`) })

	require.NoError(t, users.Create(ctx, ana, "familia"))
	got, err := users.FindByEmail(ctx, ana.Email)
	require.NoError(t, err)
	assert.Equal(t, ana, got)
	g, err := postgres.NewGuildRepository(pool).FindByMember(ctx, ana.ID)
	require.NoError(t, err)
	assert.Equal(t, "familia", g.ID)

	dup := ana
	dup.ID = "ana-create-2"
	assert.ErrorIs(t, users.Create(ctx, dup, "familia"), user.ErrAlreadyExists)

	// A duplicate id is a different constraint: not reported as a taken email.
	sameID := ana
	sameID.Email = "other@boraquest.dev"
	err = users.Create(ctx, sameID, "familia")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, user.ErrAlreadyExists)
}

func TestUserRepositoryCreateIsAtomic(t *testing.T) {
	ctx := context.Background()
	users := postgres.NewUserRepository(pool)
	ana := user.User{ID: "ana-atomic", Name: "Ana", Email: "ana-atomic@boraquest.dev", PasswordHash: "hash"}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = 'ana-atomic'`) })

	assert.ErrorIs(t, users.Create(ctx, ana, "ghost-guild"), guild.ErrNotFound)
	_, err := users.FindByEmail(ctx, ana.Email)
	assert.ErrorIs(t, err, user.ErrNotFound, "the user insert is rolled back with the membership")

	// Invalid UTF-8 fails the membership insert with an error that is not a missing guild.
	err = users.Create(ctx, ana, "\xff")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, guild.ErrNotFound)
}

func TestUserRepositoryUpdateAvatar(t *testing.T) {
	ctx := context.Background()
	users := postgres.NewUserRepository(pool)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `UPDATE users SET avatar_key = 'caio' WHERE id = 'caio'`) })

	got, err := users.UpdateAvatar(ctx, "caio", "duda")
	require.NoError(t, err)
	assert.Equal(t, "duda", got.AvatarKey)

	found, err := users.FindByEmail(ctx, "caio@boraquest.dev")
	require.NoError(t, err)
	assert.Equal(t, "duda", found.AvatarKey, "the update is persisted")

	_, err = users.UpdateAvatar(ctx, "ghost-user", "duda")
	assert.ErrorIs(t, err, user.ErrNotFound)
}

func TestUserRepositoryListByIDs(t *testing.T) {
	ctx := context.Background()
	users := postgres.NewUserRepository(pool)

	got, err := users.ListByIDs(ctx, []string{"beto", "lia", "ghost-user"})
	require.NoError(t, err)
	ids := make([]string, len(got))
	for i, u := range got {
		ids[i] = u.ID
	}
	assert.ElementsMatch(t, []string{"beto", "lia"}, ids, "unknown ids are silently skipped")

	got, err = users.ListByIDs(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, []user.User{}, got, "empty ids short-circuits without querying")
}

func TestGuildRepository(t *testing.T) {
	ctx := context.Background()
	guilds := postgres.NewGuildRepository(pool)
	familia := guild.Guild{ID: "familia", Name: "Família", UserIDs: []string{"beto", "caio", "lia", "nena"}}

	g, err := guilds.Get(ctx, "familia")
	require.NoError(t, err)
	assert.Equal(t, familia, g, "migration 0004 seeds the v1 guild with the dev users")

	g, err = guilds.FindByMember(ctx, "nena")
	require.NoError(t, err)
	assert.Equal(t, familia, g)

	_, err = guilds.Get(ctx, "ghost-guild")
	assert.ErrorIs(t, err, guild.ErrNotFound)
	_, err = guilds.FindByMember(ctx, outsiderID)
	assert.ErrorIs(t, err, guild.ErrNotMember)

	all, err := guilds.ListAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, []guild.Guild{familia}, all)
}

func TestEntryRepositorySetPhoto(t *testing.T) {
	resetEntries(t)
	ctx := context.Background()
	entries := postgres.NewEntryRepository(pool)
	now := time.Now().UTC()
	e, err := entry.New("e1", "familia", "r1", "Água", rule.ScoreTypeSum, 10, "lia", "lia", rule.FrequencyDaily, now)
	require.NoError(t, err)
	require.NoError(t, entries.Create(ctx, e))

	got, err := entries.SetPhoto(ctx, "familia", "e1", "https://cdn/e1.jpg", "pid-1")
	require.NoError(t, err)
	assert.Equal(t, "https://cdn/e1.jpg", got.PhotoURL)
	assert.Equal(t, "pid-1", got.PhotoPublicID)
	assert.Equal(t, "e1", got.ID)

	// Set at most once: a second write loses and leaves the first photo in place.
	_, err = entries.SetPhoto(ctx, "familia", "e1", "https://cdn/other.jpg", "pid-2")
	assert.ErrorIs(t, err, entry.ErrPhotoExists)
	stored, err := entries.Get(ctx, "familia", "e1")
	require.NoError(t, err)
	assert.Equal(t, "https://cdn/e1.jpg", stored.PhotoURL)
	assert.Equal(t, "pid-1", stored.PhotoPublicID)

	// Unknown id, or another guild's entry, is not found.
	_, err = entries.SetPhoto(ctx, "familia", "ghost", "u", "p")
	assert.ErrorIs(t, err, entry.ErrNotFound)
	_, err = entries.SetPhoto(ctx, "other-guild", "e1", "u", "p")
	assert.ErrorIs(t, err, entry.ErrNotFound)
}
