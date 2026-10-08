package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/guild"
)

// guildSelect reads a guild with its member ids; callers append the WHERE clause.
const guildSelect = `SELECT g.id, g.name,
	ARRAY(SELECT m.user_id FROM guild_members m WHERE m.guild_id = g.id ORDER BY m.user_id)
	FROM guilds g `

// GuildRepository reads guilds and their members from the guilds and guild_members tables.
type GuildRepository struct {
	pool *pgxpool.Pool
}

// NewGuildRepository builds a GuildRepository.
func NewGuildRepository(pool *pgxpool.Pool) *GuildRepository {
	return &GuildRepository{pool: pool}
}

// Get returns the guild with that id, or guild.ErrNotFound.
func (r *GuildRepository) Get(ctx context.Context, id string) (guild.Guild, error) {
	return r.scan(ctx, guild.ErrNotFound, guildSelect+`WHERE g.id = $1`, id)
}

// FindByMember returns the guild userID belongs to, or guild.ErrNotMember.
func (r *GuildRepository) FindByMember(ctx context.Context, userID string) (guild.Guild, error) {
	return r.scan(ctx, guild.ErrNotMember,
		guildSelect+`WHERE g.id = (SELECT guild_id FROM guild_members WHERE user_id = $1)`, userID)
}

func (r *GuildRepository) scan(ctx context.Context, notFound error, query string, arg string) (guild.Guild, error) {
	var g guild.Guild
	err := r.pool.QueryRow(ctx, query, arg).Scan(&g.ID, &g.Name, &g.UserIDs)
	if errors.Is(err, pgx.ErrNoRows) {
		return guild.Guild{}, notFound
	}
	return g, err
}
