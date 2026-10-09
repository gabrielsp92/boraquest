-- +goose Up
CREATE TABLE entries (
    id          TEXT PRIMARY KEY,
    guild_id    TEXT NOT NULL,
    rule_id     TEXT NOT NULL, -- no FK: deleting a rule must never fail or cascade (rule 6)
    rule_name   TEXT NOT NULL,
    score_type  TEXT NOT NULL CHECK (score_type IN ('sum', 'decrease')),
    points      INTEGER NOT NULL,
    member_id   TEXT NOT NULL REFERENCES users (id),
    logged_by   TEXT NOT NULL REFERENCES users (id),
    occurred_on DATE NOT NULL,
    period_key  DATE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX entries_member_occurred_idx ON entries (guild_id, member_id, occurred_on);

-- One checklist completion per rule per member per period (day or week, per rule 2).
CREATE UNIQUE INDEX entries_sum_unique_idx ON entries (rule_id, member_id, period_key) WHERE score_type = 'sum';

-- +goose Down
DROP TABLE entries;
