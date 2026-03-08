-- ============================================
-- MIGRATION 000002 - ROLLBACK
-- Restore separate date/time columns in event_dates
-- and remove timezone from venues.
-- ============================================

-- ============================================
-- PART 1: Restore event_dates columns
-- ============================================

ALTER TABLE event_dates
    ADD COLUMN event_date DATE,
    ADD COLUMN start_time TIME,
    ADD COLUMN end_time TIME;

-- Restore data from timestamps.
UPDATE event_dates
SET
    event_date = start_date_time::DATE,
    start_time = start_date_time::TIME,
    end_time   = end_date_time::TIME;

-- Restore NOT NULL on event_date.
ALTER TABLE event_dates
    ALTER COLUMN event_date SET NOT NULL;

-- Drop the check constraint.
ALTER TABLE event_dates
    DROP CONSTRAINT event_dates_check_end_after_start;

-- Drop the new columns.
ALTER TABLE event_dates
    DROP COLUMN start_date_time,
    DROP COLUMN end_date_time;

-- ============================================
-- PART 2: Remove timezone from venues
-- ============================================

ALTER TABLE venues
    DROP COLUMN timezone;
