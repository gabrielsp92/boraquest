-- +goose Up
CREATE TABLE prizes (
    guild_id   TEXT PRIMARY KEY,
    week       TEXT NOT NULL DEFAULT '' CHECK (char_length(week) <= 60),
    month      TEXT NOT NULL DEFAULT '' CHECK (char_length(month) <= 60),
    updated_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE prizes;
