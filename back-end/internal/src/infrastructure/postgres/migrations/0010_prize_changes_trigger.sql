-- +goose Up
-- 0009 reached production before the release that writes prize_changes, so edits
-- made in between were never logged, and backfilled rows carry prizes.updated_at
-- (always later than when the prize was really set). Fix both, then log every
-- prize edit with a trigger so no writer — old release or new — can skip it.

-- The oldest known prize of each slot predates the log: assume it was already in
-- effect for every earlier week rather than snapshotting those weeks with none.
UPDATE prize_changes SET changed_at = '-infinity'
WHERE id IN (SELECT min(id) FROM prize_changes GROUP BY guild_id, period);

-- Catch up on edits made while the log wasn't written.
INSERT INTO prize_changes (guild_id, period, text, changed_at)
SELECT p.guild_id, s.period, s.text, p.updated_at
FROM prizes p
CROSS JOIN LATERAL (VALUES ('week', p.week), ('month', p.month)) AS s (period, text)
WHERE s.text IS DISTINCT FROM COALESCE(
    (SELECT c.text FROM prize_changes c
     WHERE c.guild_id = p.guild_id AND c.period = s.period
     ORDER BY c.changed_at DESC, c.id DESC LIMIT 1),
    '');

-- +goose StatementBegin
CREATE FUNCTION log_prize_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF (TG_OP = 'INSERT' AND NEW.week <> '') OR (TG_OP = 'UPDATE' AND NEW.week IS DISTINCT FROM OLD.week) THEN
        INSERT INTO prize_changes (guild_id, period, text, changed_at) VALUES (NEW.guild_id, 'week', NEW.week, NEW.updated_at);
    END IF;
    IF (TG_OP = 'INSERT' AND NEW.month <> '') OR (TG_OP = 'UPDATE' AND NEW.month IS DISTINCT FROM OLD.month) THEN
        INSERT INTO prize_changes (guild_id, period, text, changed_at) VALUES (NEW.guild_id, 'month', NEW.month, NEW.updated_at);
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER prizes_log_change AFTER INSERT OR UPDATE ON prizes
FOR EACH ROW EXECUTE FUNCTION log_prize_change();

-- +goose Down
DROP TRIGGER prizes_log_change ON prizes;
DROP FUNCTION log_prize_change();
