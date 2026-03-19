-- Migration 000007: Add soft delete for volunteers and cancellation tracking for shift assignments.
--
-- Rather than deleting volunteers (which would orphan shift history),
-- we mark them inactive. A partial index keeps active-volunteer queries
-- fast without needing to touch inactive records.
--
-- Shift assignments now track cancellation time so volunteers can see
-- their full history including cancelled signups.

-- Soft delete for volunteers.
ALTER TABLE volunteers
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- Partial index: most queries only touch active volunteers.
-- This makes WHERE is_active = TRUE as fast as if inactive records didn't exist.
CREATE INDEX idx_volunteers_active ON volunteers(volunteer_id) WHERE is_active = TRUE;

-- Cancellation tracking for shift assignments.
-- NULL means the assignment is still active.
ALTER TABLE volunteer_shifts
    ADD COLUMN cancelled_at TIMESTAMP;

CREATE INDEX idx_volunteer_shifts_cancelled ON volunteer_shifts(cancelled_at) WHERE cancelled_at IS NULL;
