-- Timezone has moved to the events table (migration 000005) so that virtual
-- events can carry their own timezone. It is no longer needed on venues.

ALTER TABLE venues
    DROP COLUMN IF EXISTS timezone;
