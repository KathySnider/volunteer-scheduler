DROP INDEX IF EXISTS idx_events_recurrence_group;

ALTER TABLE events
    DROP COLUMN IF EXISTS recurrence_order,
    DROP COLUMN IF EXISTS recurrence_group_id,
    DROP COLUMN IF EXISTS timezone;
