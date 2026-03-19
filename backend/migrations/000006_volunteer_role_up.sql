-- Migration 000006: Add role-based access control.
-- Adds a role enum to volunteers and caches the role
-- in sessions to avoid repeated DB lookups on each request.

CREATE TYPE volunteer_role AS ENUM (
    'VOLUNTEER',
    'ADMINISTRATOR'
);

-- Add role to volunteers. All existing volunteers default to VOLUNTEER.
ALTER TABLE volunteers
    ADD COLUMN role volunteer_role NOT NULL DEFAULT 'VOLUNTEER';

CREATE INDEX idx_volunteers_role ON volunteers(role);

-- Cache the role in sessions so RequireAuth can populate
-- the context without an extra DB round-trip on every request.
ALTER TABLE sessions
    ADD COLUMN role volunteer_role NOT NULL DEFAULT 'VOLUNTEER';
