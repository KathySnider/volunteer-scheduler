-- Migration 000006 DOWN: Remove role-based access control.

ALTER TABLE sessions
    DROP COLUMN role;

ALTER TABLE volunteers
    DROP COLUMN role;

DROP INDEX idx_volunteers_role;

DROP TYPE volunteer_role;
