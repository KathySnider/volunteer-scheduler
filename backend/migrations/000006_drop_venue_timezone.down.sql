-- Restore timezone on venues if rolling back.
-- Defaults to America/Los_Angeles for any existing rows.

ALTER TABLE venues
    ADD COLUMN IF NOT EXISTS timezone TEXT NOT NULL DEFAULT 'America/Los_Angeles';
