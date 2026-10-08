package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

type fakeSeedRules struct {
	existing  []rule.Rule
	listErr   error
	createErr error
	created   []rule.Rule
	max       int
}

func (f *fakeSeedRules) ListByGuild(context.Context, string) ([]rule.Rule, error) {
	return f.existing, f.listErr
}

func (f *fakeSeedRules) CreateWithLimit(_ context.Context, r rule.Rule, max int) error {
	f.created, f.max = append(f.created, r), max
	return f.createErr
}

var (
	seedWater = service.RuleInput{Name: "Beber água", Frequency: rule.FrequencyDaily, ScoreType: rule.ScoreTypeSum, Score: 10}
	seedWalk  = service.RuleInput{Name: "Caminhar", Frequency: rule.FrequencyDaily, ScoreType: rule.ScoreTypeSum, Score: 15}
	seedFood  = service.RuleInput{Name: "Fast-food", Frequency: rule.FrequencyWeekly, ScoreType: rule.ScoreTypeDecrease, Score: 10}
)

func newRuleSeeder(repo *fakeSeedRules, guilds *fakeGuildFinder, users *fakeUsers) *service.RuleSeedService {
	return service.NewRuleSeedService(repo, guilds, users, fakeIDs{id: "new-id"}, fakeClock{now: ruleAt})
}

func seedInput(rules ...service.RuleInput) service.RuleSeedInput {
	return service.RuleSeedInput{GuildID: " g1 ", CreatorEmail: " LIA@boraquest.dev ", Rules: rules}
}

func TestRuleSeedServiceAddsAndSkips(t *testing.T) {
	repo := &fakeSeedRules{existing: []rule.Rule{{Name: "beber ÁGUA"}}}
	guilds, users := &fakeGuildFinder{}, &fakeUsers{user: lia}

	res, err := newRuleSeeder(repo, guilds, users).Seed(context.Background(), seedInput(seedWater, seedWalk, seedFood, seedWalk))

	require.NoError(t, err)
	assert.Equal(t, "g1", guilds.id)
	assert.Equal(t, "lia@boraquest.dev", users.email)
	assert.Equal(t, testGuild, res.Guild)
	assert.Equal(t, lia, res.Creator)
	assert.Equal(t, []string{"Beber água", "Caminhar"}, res.Skipped, "skips names the guild has (case-insensitive) and repeats in the input")
	require.Len(t, res.Added, 2)
	walk, food := res.Added[0], res.Added[1]
	assert.Equal(t, rule.Rule{
		ID: "new-id", GuildID: "g1", Name: "Caminhar", Frequency: rule.FrequencyDaily, ScoreType: rule.ScoreTypeSum,
		Score: 15, CreatedBy: "lia", CreatedAt: ruleAt.Add(time.Millisecond), UpdatedAt: ruleAt.Add(time.Millisecond),
	}, walk)
	assert.Equal(t, "Fast-food", food.Name)
	assert.True(t, food.CreatedAt.After(walk.CreatedAt), "creation times follow input order")
	assert.Equal(t, res.Added, repo.created)
	assert.Equal(t, rule.MaxPerGuild, repo.max)
}

func TestRuleSeedServiceDryRun(t *testing.T) {
	repo := &fakeSeedRules{}
	in := seedInput(seedWater, seedWalk)
	in.DryRun = true

	res, err := newRuleSeeder(repo, &fakeGuildFinder{}, &fakeUsers{user: lia}).Seed(context.Background(), in)

	require.NoError(t, err)
	assert.Len(t, res.Added, 2)
	assert.Empty(t, repo.created, "a dry run must not write")
}

func TestRuleSeedServiceErrors(t *testing.T) {
	outsider := user.User{ID: "tom", Email: "tom@boraquest.dev"}
	full := make([]rule.Rule, rule.MaxPerGuild-1)
	cases := map[string]struct {
		repo   *fakeSeedRules
		guilds *fakeGuildFinder
		users  *fakeUsers
		rules  []service.RuleInput
		want   error
	}{
		"unknown guild":     {&fakeSeedRules{}, &fakeGuildFinder{err: guild.ErrNotFound}, &fakeUsers{user: lia}, []service.RuleInput{seedWater}, guild.ErrNotFound},
		"unknown creator":   {&fakeSeedRules{}, &fakeGuildFinder{}, &fakeUsers{err: user.ErrNotFound}, []service.RuleInput{seedWater}, user.ErrNotFound},
		"creator not in it": {&fakeSeedRules{}, &fakeGuildFinder{}, &fakeUsers{user: outsider}, []service.RuleInput{seedWater}, guild.ErrNotMember},
		"list fails":        {&fakeSeedRules{listErr: errBoom}, &fakeGuildFinder{}, &fakeUsers{user: lia}, []service.RuleInput{seedWater}, errBoom},
		"invalid rule":      {&fakeSeedRules{}, &fakeGuildFinder{}, &fakeUsers{user: lia}, []service.RuleInput{seedWater, {Name: "x", Frequency: "monthly", ScoreType: rule.ScoreTypeSum, Score: 1}}, rule.ErrInvalidFrequency},
		"over the limit":    {&fakeSeedRules{existing: full}, &fakeGuildFinder{}, &fakeUsers{user: lia}, []service.RuleInput{seedWater, seedWalk}, rule.ErrLimitReached},
		"create fails":      {&fakeSeedRules{createErr: errBoom}, &fakeGuildFinder{}, &fakeUsers{user: lia}, []service.RuleInput{seedWater, seedWalk}, errBoom},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := newRuleSeeder(tc.repo, tc.guilds, tc.users).Seed(context.Background(), seedInput(tc.rules...))

			assert.ErrorIs(t, err, tc.want)
			if tc.repo.createErr == nil {
				assert.Empty(t, tc.repo.created, "nothing is written when validation fails")
			}
		})
	}
}
