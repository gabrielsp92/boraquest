-- +goose Up
ALTER TABLE entries ADD COLUMN photo_url TEXT NOT NULL DEFAULT '';
ALTER TABLE entries ADD COLUMN photo_public_id TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE entries DROP COLUMN photo_public_id;
ALTER TABLE entries DROP COLUMN photo_url;
