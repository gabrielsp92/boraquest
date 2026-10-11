-- +goose Up
ALTER TABLE users ADD COLUMN avatar_key TEXT NOT NULL DEFAULT 'lia'
    CHECK (avatar_key IN ('beto', 'caio', 'duda', 'lia', 'nena', 'tom'));

-- Preserve today's look for the four seeded dev users (Business rule 4).
UPDATE users SET avatar_key = id WHERE id IN ('beto', 'caio', 'lia', 'nena');

-- +goose Down
ALTER TABLE users DROP COLUMN avatar_key;
