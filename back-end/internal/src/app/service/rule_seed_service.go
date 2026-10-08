package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

// RuleSeedRepository lists and stores a guild's rules.
type RuleSeedRepository interface {
	ListByGuild(ctx context.Context, guildID string) ([]rule.Rule, error)
	// CreateWithLimit returns rule.ErrLimitReached when the guild already holds max rules.
	CreateWithLimit(ctx context.Context, r rule.Rule, max int) error
}

// RuleSeedInput describes the rules to add to a guild and the member recorded
// as their creator. DryRun validates and checks the database without writing.
type RuleSeedInput struct {
	GuildID      string
	CreatorEmail string
	Rules        []RuleInput
	DryRun       bool
}

// RuleSeedResult lists the rules added (in a dry run, the ones that would be)
// and the names skipped because the guild, or an earlier entry, already has them.
type RuleSeedResult struct {
	Guild   guild.Guild
	Creator user.User
	Added   []rule.Rule
	Skipped []string
}

// RuleSeedService adds a fixed set of rules to a guild, e.g. from a seed file.
type RuleSeedService struct {
	rules  RuleSeedRepository
	guilds GuildFinder
	users  UserRepository
	ids    IDGenerator
	clock  Clock
}

// NewRuleSeedService builds a RuleSeedService.
func NewRuleSeedService(rules RuleSeedRepository, guilds GuildFinder, users UserRepository, ids IDGenerator, clock Clock) *RuleSeedService {
	return &RuleSeedService{rules: rules, guilds: guilds, users: users, ids: ids, clock: clock}
}

// Seed adds every rule whose name (case-insensitive) the guild does not have
// yet, so running it again is a no-op. All rules are validated, and the guild
// limit checked, before anything is written. Creation times are spaced one
// millisecond apart so the guild lists them in input order.
func (s *RuleSeedService) Seed(ctx context.Context, in RuleSeedInput) (RuleSeedResult, error) {
	g, err := s.guilds.Get(ctx, strings.TrimSpace(in.GuildID))
	if err != nil {
		return RuleSeedResult{}, err
	}
	creator, err := s.users.FindByEmail(ctx, user.NormalizeEmail(in.CreatorEmail))
	if err != nil {
		return RuleSeedResult{}, err
	}
	if !g.HasMember(creator.ID) {
		return RuleSeedResult{}, guild.ErrNotMember
	}
	existing, err := s.rules.ListByGuild(ctx, g.ID)
	if err != nil {
		return RuleSeedResult{}, err
	}
	taken := make(map[string]bool, len(existing))
	for _, r := range existing {
		taken[strings.ToLower(r.Name)] = true
	}

	res := RuleSeedResult{Guild: g, Creator: creator}
	now := s.clock.Now()
	for i, x := range in.Rules {
		r, err := rule.New(s.ids.NewID(), g.ID, creator.ID, x.Name, x.Frequency, x.ScoreType, x.Score, now.Add(time.Duration(i)*time.Millisecond))
		if err != nil {
			return RuleSeedResult{}, fmt.Errorf("rule %d (%q): %w", i+1, x.Name, err)
		}
		key := strings.ToLower(r.Name)
		if taken[key] {
			res.Skipped = append(res.Skipped, r.Name)
			continue
		}
		taken[key] = true
		res.Added = append(res.Added, r)
	}
	if len(existing)+len(res.Added) > rule.MaxPerGuild {
		return RuleSeedResult{}, rule.ErrLimitReached
	}
	if in.DryRun {
		return res, nil
	}
	for _, r := range res.Added {
		if err := s.rules.CreateWithLimit(ctx, r, rule.MaxPerGuild); err != nil {
			return RuleSeedResult{}, err
		}
	}
	return res, nil
}
