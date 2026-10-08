// Package guild provides guild lookups. For v1 the only guild is hardcoded here;
// it moves to the database once guilds can be managed.
package guild

import (
	"context"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
)

// Default is the single v1 guild. Its members are the users seeded by the
// 0003_seed_dev_users migration.
var Default = guild.Guild{
	ID:      "familia",
	Name:    "Família",
	UserIDs: []string{"lia", "beto", "nena", "caio"},
}

// StaticRepository serves guilds from a fixed in-memory list.
type StaticRepository struct {
	guilds []guild.Guild
}

// NewStaticRepository builds a StaticRepository over guilds.
func NewStaticRepository(guilds ...guild.Guild) *StaticRepository {
	return &StaticRepository{guilds: guilds}
}

// FindByMember returns the first guild userID belongs to.
func (r *StaticRepository) FindByMember(_ context.Context, userID string) (guild.Guild, error) {
	for _, g := range r.guilds {
		if g.HasMember(userID) {
			return g, nil
		}
	}
	return guild.Guild{}, guild.ErrNotMember
}
