-- Migration 000005 down: Remove volunteer_id from sessions table.

DROP INDEX IF EXISTS idx_sessions_volunteer_id;

ALTER TABLE sessions
    DROP COLUMN IF EXISTS volunteer_id;