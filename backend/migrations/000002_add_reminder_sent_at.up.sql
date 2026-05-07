-- ============================================================================
-- Add reminder_sent_at to volunteer_shifts
--
-- Tracks when a 24-hour shift reminder was sent to each volunteer.
-- NULL means no reminder has been sent yet.
-- Using TIMESTAMPTZ so times are stored correctly regardless of server timezone.
-- ============================================================================

ALTER TABLE volunteer_shifts
    ADD COLUMN reminder_sent_at TIMESTAMPTZ;

-- Partial index to make the reminder query fast:
-- only indexes rows that still need a reminder sent.
CREATE INDEX idx_volunteer_shifts_reminder
    ON volunteer_shifts(shift_id)
    WHERE reminder_sent_at IS NULL
      AND cancelled_at IS NULL;
