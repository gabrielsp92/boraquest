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
