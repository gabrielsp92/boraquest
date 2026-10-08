-- Guilds move from code (the old infrastructure/guild static list) to the
-- database. For v1 a user belongs to at most one guild, hence user_id is the key.

-- +goose Up
CREATE TABLE guilds (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE guild_members (
    user_id  TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE
);

CREATE INDEX guild_members_guild_id_idx ON guild_members (guild_id);

-- The v1 guild, with whichever 0003 dev users exist.
INSERT INTO guilds (id, name) VALUES ('familia', 'Família');
INSERT INTO guild_members (user_id, guild_id)
    SELECT id, 'familia' FROM users WHERE id IN ('lia', 'beto', 'nena', 'caio');

-- +goose Down
DROP TABLE guild_members;
DROP TABLE guilds;
