// Package guild holds the domain model for guilds: groups of users that share rules.
package guild

import (
	"errors"
	"slices"
)

var (
	// ErrNotMember is returned when a user does not belong to any guild.
	ErrNotMember = errors.New("guild: user is not a member")
	// ErrNotFound is returned when no guild has the requested id.
	ErrNotFound = errors.New("guild: not found")
)

// Guild is a group of users.
type Guild struct {
	ID      string
	Name    string
	UserIDs []string
}

// HasMember reports whether userID belongs to g.
func (g Guild) HasMember(userID string) bool {
	return slices.Contains(g.UserIDs, userID)
}
