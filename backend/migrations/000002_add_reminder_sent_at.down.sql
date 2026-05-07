-- ============================================================================
-- Revert: remove reminder_sent_at from volunteer_shifts
-- ============================================================================

DROP INDEX IF EXISTS idx_volunteer_shifts_reminder;

ALTER TABLE volunteer_shifts
    DROP COLUMN IF EXISTS reminder_sent_at;
