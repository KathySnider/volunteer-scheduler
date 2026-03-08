-- ============================================
-- MIGRATION 000002 - Consolidate event_dates 
-- date/time columns into timestamps, and add
-- timezone to venues.
-- ============================================

-- ============================================
-- PART 1: Add timezone to venues
-- ============================================

ALTER TABLE venues
    ADD COLUMN timezone TEXT NOT NULL DEFAULT 'America/Los_Angeles';

-- Update existing venues with correct timezones.
UPDATE venues SET timezone = 'America/Los_Angeles' WHERE city IN ('Las Vegas', 'Henderson', 'N. Las Vegas');

-- ============================================
-- PART 2: Consolidate event_dates columns
-- ============================================

-- Combine the separate date, start_time, end_time columns
-- into start_date_time and end_date_time TIMESTAMP columns.
-- This makes event_dates consistent with shifts and supports
-- UTC storage.

ALTER TABLE event_dates
    ADD COLUMN start_date_time TIMESTAMP,
    ADD COLUMN end_date_time TIMESTAMP;

-- Populate new columns from existing data.
UPDATE event_dates
SET
    start_date_time = (event_date + start_time),
    end_date_time   = (event_date + end_time);

-- Now make them NOT NULL.
ALTER TABLE event_dates
    ALTER COLUMN start_date_time SET NOT NULL,
    ALTER COLUMN end_date_time SET NOT NULL;

-- Add a check constraint consistent with shifts.
ALTER TABLE event_dates
    ADD CONSTRAINT event_dates_check_end_after_start
    CHECK (end_date_time > start_date_time);

-- Drop the old columns.
ALTER TABLE event_dates
    DROP COLUMN event_date,
    DROP COLUMN start_time,
    DROP COLUMN end_time;
