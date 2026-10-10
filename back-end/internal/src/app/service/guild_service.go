package service

import (
	"context"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/user"
)

// GuildUserRepository looks up users by id.
type GuildUserRepository interface {
	ListByIDs(ctx context.Context, ids []string) ([]user.User, error)
}

// GuildMember is a lightweight view of a guild member.
type GuildMember struct {
	ID   string
	Name string
}

// GuildService resolves information about the caller's guild.
type GuildService struct {
	guilds GuildRepository // reused interface from rule_service.go
	users  GuildUserRepository
}

// NewGuildService builds a GuildService.
func NewGuildService(guilds GuildRepository, users GuildUserRepository) *GuildService {
	return &GuildService{guilds: guilds, users: users}
}

// ListMembers returns the members of callerID's guild, in the guild's member order.
func (s *GuildService) ListMembers(ctx context.Context, callerID string) ([]GuildMember, error) {
	g, err := s.guilds.FindByMember(ctx, callerID)
	if err != nil {
		return nil, err
	}
	users, err := s.users.ListByIDs(ctx, g.UserIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]user.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	members := make([]GuildMember, 0, len(g.UserIDs))
	for _, id := range g.UserIDs {
		if u, ok := byID[id]; ok {
			members = append(members, GuildMember{ID: u.ID, Name: u.Name})
		}
	}
	return members, nil
}
