package unit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
)

type fakeGuilds struct {
	guild guild.Guild
	err   error
}

func (f fakeGuilds) FindByMember(context.Context, string) (guild.Guild, error) { return f.guild, f.err }

type fakeRuleRepo struct {
	listed    []rule.Rule
	got       rule.Rule
	getErr    error
	createErr error
	updateErr error
	deleteErr error
	listErr   error

	created   rule.Rule
	createMax int
	updated   rule.Rule
	getArgs   [2]string
	deleteArg [2]string
	listArg   string
}

func (f *fakeRuleRepo) ListByGuild(_ context.Context, guildID string) ([]rule.Rule, error) {
	f.listArg = guildID
	return f.listed, f.listErr
}

func (f *fakeRuleRepo) Get(_ context.Context, guildID, id string) (rule.Rule, error) {
	f.getArgs = [2]string{guildID, id}
	return f.got, f.getErr
}

func (f *fakeRuleRepo) CreateWithLimit(_ context.Context, r rule.Rule, max int) error {
	f.created, f.createMax = r, max
	return f.createErr
}

func (f *fakeRuleRepo) Update(_ context.Context, r rule.Rule) error {
	f.updated = r
	return f.updateErr
}

func (f *fakeRuleRepo) Delete(_ context.Context, guildID, id string) error {
	f.deleteArg = [2]string{guildID, id}
	return f.deleteErr
}

type fakeIDs struct{ id string }

func (f fakeIDs) NewID() string { return f.id }

var (
	testGuild = guild.Guild{ID: "g1", Name: "Família", UserIDs: []string{"lia"}}
	ruleAt    = time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	errBoom   = errors.New("boom")
)

func newRuleService(repo *fakeRuleRepo, guilds fakeGuilds) *service.RuleService {
	return service.NewRuleService(repo, guilds, fakeIDs{id: "new-id"}, fakeClock{now: ruleAt})
}

func validInput() service.RuleInput {
	return service.RuleInput{Name: "Água", Frequency: rule.FrequencyDaily, ScoreType: rule.ScoreTypeSum, Score: 10}
}

func TestRuleServiceList(t *testing.T) {
	repo := &fakeRuleRepo{listed: []rule.Rule{{ID: "r1"}}}

	rules, err := newRuleService(repo, fakeGuilds{guild: testGuild}).List(context.Background(), "lia")

	require.NoError(t, err)
	assert.Equal(t, []rule.Rule{{ID: "r1"}}, rules)
	assert.Equal(t, "g1", repo.listArg)
}

func TestRuleServiceGet(t *testing.T) {
	repo := &fakeRuleRepo{got: rule.Rule{ID: "r1"}}

	r, err := newRuleService(repo, fakeGuilds{guild: testGuild}).Get(context.Background(), "lia", "r1")

	require.NoError(t, err)
	assert.Equal(t, "r1", r.ID)
	assert.Equal(t, [2]string{"g1", "r1"}, repo.getArgs)
}

func TestRuleServiceCreate(t *testing.T) {
	repo := &fakeRuleRepo{}

	r, err := newRuleService(repo, fakeGuilds{guild: testGuild}).Create(context.Background(), "lia", validInput())

	require.NoError(t, err)
	want := rule.Rule{
		ID: "new-id", GuildID: "g1", Name: "Água", Frequency: rule.FrequencyDaily, ScoreType: rule.ScoreTypeSum,
		Score: 10, CreatedBy: "lia", CreatedAt: ruleAt, UpdatedAt: ruleAt,
	}
	assert.Equal(t, want, r)
	assert.Equal(t, want, repo.created)
	assert.Equal(t, rule.MaxPerGuild, repo.createMax)
}

func TestRuleServiceCreateInvalid(t *testing.T) {
	repo := &fakeRuleRepo{}
	in := validInput()
	in.Score = 0

	_, err := newRuleService(repo, fakeGuilds{guild: testGuild}).Create(context.Background(), "lia", in)

	assert.ErrorIs(t, err, rule.ErrInvalidScore)
	assert.Empty(t, repo.created.ID)
}

func TestRuleServiceCreateRepoError(t *testing.T) {
	repo := &fakeRuleRepo{createErr: rule.ErrLimitReached}

	_, err := newRuleService(repo, fakeGuilds{guild: testGuild}).Create(context.Background(), "lia", validInput())

	assert.ErrorIs(t, err, rule.ErrLimitReached)
}

func TestRuleServiceUpdate(t *testing.T) {
	existing, err := rule.New("r1", "g1", "beto", "Água", rule.FrequencyDaily, rule.ScoreTypeSum, 10, ruleAt.Add(-time.Hour))
	require.NoError(t, err)
	repo := &fakeRuleRepo{got: existing}
	in := service.RuleInput{Name: "Fast-food", Frequency: rule.FrequencyWeekly, ScoreType: rule.ScoreTypeDecrease, Score: 20}

	r, err := newRuleService(repo, fakeGuilds{guild: testGuild}).Update(context.Background(), "lia", "r1", in)

	require.NoError(t, err)
	assert.Equal(t, [2]string{"g1", "r1"}, repo.getArgs)
	assert.Equal(t, "Fast-food", r.Name)
	assert.Equal(t, "beto", r.CreatedBy)
	assert.Equal(t, ruleAt, r.UpdatedAt)
	assert.Equal(t, r, repo.updated)
}

func TestRuleServiceUpdateGetError(t *testing.T) {
	repo := &fakeRuleRepo{getErr: rule.ErrNotFound}

	_, err := newRuleService(repo, fakeGuilds{guild: testGuild}).Update(context.Background(), "lia", "r1", validInput())

	assert.ErrorIs(t, err, rule.ErrNotFound)
}

func TestRuleServiceUpdateInvalid(t *testing.T) {
	repo := &fakeRuleRepo{got: rule.Rule{ID: "r1", GuildID: "g1"}}
	in := validInput()
	in.Frequency = "monthly"

	_, err := newRuleService(repo, fakeGuilds{guild: testGuild}).Update(context.Background(), "lia", "r1", in)

	assert.ErrorIs(t, err, rule.ErrInvalidFrequency)
	assert.Empty(t, repo.updated.ID)
}

func TestRuleServiceUpdateRepoError(t *testing.T) {
	repo := &fakeRuleRepo{got: rule.Rule{ID: "r1", GuildID: "g1"}, updateErr: errBoom}

	_, err := newRuleService(repo, fakeGuilds{guild: testGuild}).Update(context.Background(), "lia", "r1", validInput())

	assert.ErrorIs(t, err, errBoom)
}

func TestRuleServiceDelete(t *testing.T) {
	repo := &fakeRuleRepo{}

	err := newRuleService(repo, fakeGuilds{guild: testGuild}).Delete(context.Background(), "lia", "r1")

	require.NoError(t, err)
	assert.Equal(t, [2]string{"g1", "r1"}, repo.deleteArg)
}

func TestRuleServiceRejectsNonMembers(t *testing.T) {
	repo := &fakeRuleRepo{}
	svc := newRuleService(repo, fakeGuilds{err: guild.ErrNotMember})
	ctx := context.Background()

	_, err := svc.List(ctx, "x")
	assert.ErrorIs(t, err, guild.ErrNotMember)
	_, err = svc.Get(ctx, "x", "r1")
	assert.ErrorIs(t, err, guild.ErrNotMember)
	_, err = svc.Create(ctx, "x", validInput())
	assert.ErrorIs(t, err, guild.ErrNotMember)
	_, err = svc.Update(ctx, "x", "r1", validInput())
	assert.ErrorIs(t, err, guild.ErrNotMember)
	assert.ErrorIs(t, svc.Delete(ctx, "x", "r1"), guild.ErrNotMember)

	assert.Equal(t, fakeRuleRepo{}, *repo, "repository must not be touched")
}
