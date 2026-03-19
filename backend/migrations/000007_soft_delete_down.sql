-- Migration 000007 DOWN: Remove soft delete and cancellation tracking.

DROP INDEX idx_volunteer_shifts_cancelled;

ALTER TABLE volunteer_shifts
    DROP COLUMN cancelled_at;

DROP INDEX idx_volunteers_active;

ALTER TABLE volunteers
    DROP COLUMN is_active;
