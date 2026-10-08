package integration_test

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/clock"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/idgen"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/postgres"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/seedfile"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
)

// newRuleSeeder wires the seed service like cmd/seedrules/main.go, against the test database.
func newRuleSeeder() *service.RuleSeedService {
	return service.NewRuleSeedService(
		postgres.NewRuleRepository(pool),
		postgres.NewGuildRepository(pool),
		postgres.NewUserRepository(pool),
		idgen.UUIDGenerator{},
		clock.SystemClock{},
	)
}

// Seeding seeds/rules.json into familia: a dry run writes nothing, the real run
// makes the rules visible through the API in file order, and re-running skips them all.
func TestSeedRulesFromFile(t *testing.T) {
	resetRules(t)
	t.Cleanup(func() { resetRules(t) })
	ctx := context.Background()
	f, err := os.Open("../../seeds/rules.json")
	require.NoError(t, err)
	rules, err := seedfile.ReadRules(f)
	f.Close()
	require.NoError(t, err)
	in := service.RuleSeedInput{GuildID: "familia", CreatorEmail: "nena@boraquest.dev", Rules: rules, DryRun: true}
	srv := newServer(t)
	token := login(t, srv.URL, "lia@boraquest.dev")
	listRules := func() []controllers.RuleResponse {
		res := call(t, http.MethodGet, srv.URL+"/api/v1/rules", token, nil)
		require.Equal(t, http.StatusOK, res.StatusCode)
		return decodeBody[controllers.RuleListResponse](t, res).Rules
	}

	dry, err := newRuleSeeder().Seed(ctx, in)
	require.NoError(t, err)
	assert.Len(t, dry.Added, len(rules))
	assert.Empty(t, listRules(), "dry run must not write")

	in.DryRun = false
	res, err := newRuleSeeder().Seed(ctx, in)
	require.NoError(t, err)
	assert.Len(t, res.Added, len(rules))
	listed := listRules()
	require.Len(t, listed, len(rules))
	for i, x := range rules {
		assert.Equal(t, x.Name, listed[i].Name)
		assert.Equal(t, string(x.ScoreType), listed[i].ScoreType)
		assert.Equal(t, x.Score, listed[i].Score)
		assert.Equal(t, "nena", listed[i].CreatedBy)
	}

	again, err := newRuleSeeder().Seed(ctx, in)
	require.NoError(t, err)
	assert.Empty(t, again.Added)
	assert.Len(t, again.Skipped, len(rules))
	assert.Len(t, listRules(), len(rules))
}
