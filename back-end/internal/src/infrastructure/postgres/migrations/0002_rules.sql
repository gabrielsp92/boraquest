-- +goose Up
CREATE TABLE rules (
    id         TEXT PRIMARY KEY,
    guild_id   TEXT NOT NULL,
    name       TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 32),
    frequency  TEXT NOT NULL CHECK (frequency IN ('daily', 'weekly')),
    score_type TEXT NOT NULL CHECK (score_type IN ('sum', 'decrease')),
    score      INTEGER NOT NULL CHECK (score BETWEEN 1 AND 100),
    created_by TEXT NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX rules_guild_id_idx ON rules (guild_id, created_at);

-- +goose Down
DROP TABLE rules;
