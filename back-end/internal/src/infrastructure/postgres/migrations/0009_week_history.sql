-- +goose Up
-- Append-only log of every prize edit, so a closed week can snapshot the prize
-- that was in effect when it ended, even if it is closed (lazily) later on.
CREATE TABLE prize_changes (
    id         BIGSERIAL PRIMARY KEY,
    guild_id   TEXT NOT NULL,
    period     TEXT NOT NULL CHECK (period IN ('week', 'month')),
    text       TEXT NOT NULL CHECK (char_length(text) <= 60),
    changed_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX prize_changes_lookup_idx ON prize_changes (guild_id, period, changed_at);

-- Best known history for prizes set before this log existed.
INSERT INTO prize_changes (guild_id, period, text, changed_at)
SELECT guild_id, 'week', week, updated_at FROM prizes WHERE week <> '';
INSERT INTO prize_changes (guild_id, period, text, changed_at)
SELECT guild_id, 'month', month, updated_at FROM prizes WHERE month <> '';

-- One row per closed week (Monday..Sunday, America/Sao_Paulo) that had entries.
CREATE TABLE week_results (
    guild_id   TEXT NOT NULL,
    week_start DATE NOT NULL,
    week_end   DATE NOT NULL,
    prize      TEXT NOT NULL DEFAULT '',
    closed_at  TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (guild_id, week_start)
);

-- Frozen standings of a closed week; rank 1 is the winner.
CREATE TABLE week_result_standings (
    guild_id   TEXT NOT NULL,
    week_start DATE NOT NULL,
    member_id  TEXT NOT NULL REFERENCES users (id),
    rank       INTEGER NOT NULL,
    points     INTEGER NOT NULL,
    completed  INTEGER NOT NULL,
    PRIMARY KEY (guild_id, week_start, member_id),
    FOREIGN KEY (guild_id, week_start) REFERENCES week_results (guild_id, week_start) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE week_result_standings;
DROP TABLE week_results;
DROP TABLE prize_changes;
