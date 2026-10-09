package service

import (
	"context"
	"time"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/prize"
)

// PrizeRepository persists a guild's prizes, always scoped to the guild.
type PrizeRepository interface {
	// Get returns the guild's prizes, or a zero Prizes{GuildID: guildID} if none has ever been set.
	Get(ctx context.Context, guildID string) (prize.Prizes, error)
	// Upsert creates the guild's row if missing, then sets just the given period's column.
	Upsert(ctx context.Context, guildID string, period prize.Period, text string, now time.Time) (prize.Prizes, error)
}

// PrizeService manages the prizes of the caller's guild.
type PrizeService struct {
	prizes PrizeRepository
	guilds GuildRepository // reused from rule_service.go
	clock  Clock
}

// NewPrizeService builds a PrizeService.
func NewPrizeService(prizes PrizeRepository, guilds GuildRepository, clock Clock) *PrizeService {
	return &PrizeService{prizes: prizes, guilds: guilds, clock: clock}
}

// Get returns the prizes of callerID's guild.
func (s *PrizeService) Get(ctx context.Context, callerID string) (prize.Prizes, error) {
	g, err := s.guilds.FindByMember(ctx, callerID)
	if err != nil {
		return prize.Prizes{}, err
	}
	return s.prizes.Get(ctx, g.ID)
}

// Set replaces one of callerID's guild's prize slots.
func (s *PrizeService) Set(ctx context.Context, callerID string, period prize.Period, text string) (prize.Prizes, error) {
	g, err := s.guilds.FindByMember(ctx, callerID)
	if err != nil {
		return prize.Prizes{}, err
	}
	clean, err := prize.Clean(text)
	if err != nil {
		return prize.Prizes{}, err
	}
	return s.prizes.Upsert(ctx, g.ID, period, clean, s.clock.Now())
}
