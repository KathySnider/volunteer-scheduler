-- Migration 000005: Add volunteer_id to sessions table.
-- This allows session validation to return the volunteer's ID
-- rather than their email, so email changes don't break sessions.

ALTER TABLE sessions
    ADD COLUMN volunteer_id INTEGER REFERENCES volunteers(volunteer_id) ON DELETE CASCADE;

CREATE INDEX idx_sessions_volunteer_id ON sessions(volunteer_id);