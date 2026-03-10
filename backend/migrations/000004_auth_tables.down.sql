-- Migration 000004 down: Remove auth tables and volunteer login fields.

ALTER TABLE volunteers
    DROP COLUMN IF EXISTS last_login_at,
    DROP COLUMN IF EXISTS created_at;

DROP INDEX IF EXISTS idx_sessions_expires_at;
DROP INDEX IF EXISTS idx_sessions_email;
DROP INDEX IF EXISTS idx_sessions_token;
DROP TABLE IF EXISTS sessions;

DROP INDEX IF EXISTS idx_magic_links_email;
DROP INDEX IF EXISTS idx_magic_links_token;
DROP TABLE IF EXISTS magic_links;
