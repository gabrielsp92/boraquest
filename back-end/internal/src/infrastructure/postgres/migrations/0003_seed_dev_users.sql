-- Development users for v1, matching the hardcoded guild in
-- infrastructure/guild. Every password is "boraquest-dev".
-- Not production-safe: replace with real sign-up before going live.

-- +goose Up
INSERT INTO users (id, name, email, password_hash) VALUES
    ('lia',  'Lia',     'lia@boraquest.dev',  '$2a$10$Gp4jLSGUte5GohMyY5CNneJzHeChDbUBNpF3xxz69XFyXMLq1SSRi'),
    ('beto', 'Beto',    'beto@boraquest.dev', '$2a$10$Gp4jLSGUte5GohMyY5CNneJzHeChDbUBNpF3xxz69XFyXMLq1SSRi'),
    ('nena', 'Vó Nena', 'nena@boraquest.dev', '$2a$10$Gp4jLSGUte5GohMyY5CNneJzHeChDbUBNpF3xxz69XFyXMLq1SSRi'),
    ('caio', 'Caio',    'caio@boraquest.dev', '$2a$10$Gp4jLSGUte5GohMyY5CNneJzHeChDbUBNpF3xxz69XFyXMLq1SSRi');

-- +goose Down
DELETE FROM users WHERE id IN ('lia', 'beto', 'nena', 'caio');
