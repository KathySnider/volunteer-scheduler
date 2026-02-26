-- Rollback sessions table
DROP INDEX IF EXISTS idx_sessions_expires_at;
DROP INDEX IF EXISTS idx_sessions_email;
DROP INDEX IF EXISTS idx_sessions_token;
DROP TABLE IF EXISTS sessions;
