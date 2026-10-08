package service

import (
	"context"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
)

// RuleRepository persists rules, always scoped to a guild.
type RuleRepository interface {
	ListByGuild(ctx context.Context, guildID string) ([]rule.Rule, error)
	// Get returns rule.ErrNotFound when no rule with id exists in the guild.
	Get(ctx context.Context, guildID, id string) (rule.Rule, error)
	// CreateWithLimit stores r unless its guild already holds max rules, in which
	// case it returns rule.ErrLimitReached. The check and insert must be atomic.
	CreateWithLimit(ctx context.Context, r rule.Rule, max int) error
	// Update returns rule.ErrNotFound when the rule does not exist in its guild.
	Update(ctx context.Context, r rule.Rule) error
	// Delete returns rule.ErrNotFound when no rule with id exists in the guild.
	Delete(ctx context.Context, guildID, id string) error
}

// GuildRepository looks up guilds.
type GuildRepository interface {
	// FindByMember returns guild.ErrNotMember when userID belongs to no guild.
	FindByMember(ctx context.Context, userID string) (guild.Guild, error)
}

// IDGenerator creates unique identifiers for new entities.
type IDGenerator interface {
	NewID() string
}

// RuleInput holds the editable fields of a rule.
type RuleInput struct {
	Name      string
	Frequency rule.Frequency
	ScoreType rule.ScoreType
	Score     int
}

// RuleService manages the rules of the caller's guild.
type RuleService struct {
	rules  RuleRepository
	guilds GuildRepository
	ids    IDGenerator
	clock  Clock
}

// NewRuleService builds a RuleService.
func NewRuleService(rules RuleRepository, guilds GuildRepository, ids IDGenerator, clock Clock) *RuleService {
	return &RuleService{rules: rules, guilds: guilds, ids: ids, clock: clock}
}

// List returns every rule of userID's guild.
func (s *RuleService) List(ctx context.Context, userID string) ([]rule.Rule, error) {
	g, err := s.guilds.FindByMember(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.rules.ListByGuild(ctx, g.ID)
}

// Get returns one rule of userID's guild.
func (s *RuleService) Get(ctx context.Context, userID, id string) (rule.Rule, error) {
	g, err := s.guilds.FindByMember(ctx, userID)
	if err != nil {
		return rule.Rule{}, err
	}
	return s.rules.Get(ctx, g.ID, id)
}

// Create adds a rule to userID's guild, enforcing rule.MaxPerGuild.
func (s *RuleService) Create(ctx context.Context, userID string, in RuleInput) (rule.Rule, error) {
	g, err := s.guilds.FindByMember(ctx, userID)
	if err != nil {
		return rule.Rule{}, err
	}
	r, err := rule.New(s.ids.NewID(), g.ID, userID, in.Name, in.Frequency, in.ScoreType, in.Score, s.clock.Now())
	if err != nil {
		return rule.Rule{}, err
	}
	if err := s.rules.CreateWithLimit(ctx, r, rule.MaxPerGuild); err != nil {
		return rule.Rule{}, err
	}
	return r, nil
}

// Update replaces the editable fields of a rule in userID's guild.
func (s *RuleService) Update(ctx context.Context, userID, id string, in RuleInput) (rule.Rule, error) {
	g, err := s.guilds.FindByMember(ctx, userID)
	if err != nil {
		return rule.Rule{}, err
	}
	r, err := s.rules.Get(ctx, g.ID, id)
	if err != nil {
		return rule.Rule{}, err
	}
	if err := r.Update(in.Name, in.Frequency, in.ScoreType, in.Score, s.clock.Now()); err != nil {
		return rule.Rule{}, err
	}
	if err := s.rules.Update(ctx, r); err != nil {
		return rule.Rule{}, err
	}
	return r, nil
}

// Delete removes a rule from userID's guild.
func (s *RuleService) Delete(ctx context.Context, userID, id string) error {
	g, err := s.guilds.FindByMember(ctx, userID)
	if err != nil {
		return err
	}
	return s.rules.Delete(ctx, g.ID, id)
}
