-- ============================================================================
-- MIGRATION 000002: Replace job_type enum with job_types lookup table
--
-- Rationale: job types need to be admin-editable (e.g. adding drivers,
-- setup/cleanup crews) without a schema change. This follows the same
-- pattern as the existing service_types table.
--
-- The old job_type enum values map to the new table as follows:
--   event_support  → event_support
--   advocacy       → advocacy
--   speaker        → speaker
--   volunteer_lead → volunteer_lead
--   attendee_only  → attendee_only
--   other          → other
--
-- The CHECK constraint on opportunities.other_job_description is also
-- removed: with editable job types, "other" is no longer a special case.
-- Admins can simply create a descriptive job type instead.
-- ============================================================================


-- ----------------------------------------------------------------------------
-- 1. Create the job_types lookup table
-- ----------------------------------------------------------------------------

CREATE TABLE job_types (
    job_type_id SERIAL PRIMARY KEY,
    code        VARCHAR(50) UNIQUE NOT NULL,
    name        TEXT NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order  INT NOT NULL DEFAULT 0,
    CHECK (code = lower(code))
);

CREATE INDEX idx_job_types_active ON job_types(job_type_id) WHERE is_active = TRUE;


-- ----------------------------------------------------------------------------
-- 2. Seed with the existing enum values (preserves event history)
-- ----------------------------------------------------------------------------

INSERT INTO job_types (code, name, sort_order) VALUES
    ('event_support',  'Event Support',   10),
    ('advocacy',       'Advocacy',        20),
    ('speaker',        'Speaker',         30),
    ('volunteer_lead', 'Volunteer Lead',  40),
    ('attendee_only',  'Attendee Only',   50),
    ('other',          'Other',           60);


-- ----------------------------------------------------------------------------
-- 3. Add the new FK column to opportunities (nullable during migration)
-- ----------------------------------------------------------------------------

ALTER TABLE opportunities
    ADD COLUMN job_type_id INT REFERENCES job_types(job_type_id) ON DELETE RESTRICT;


-- ----------------------------------------------------------------------------
-- 4. Populate job_type_id from the existing job enum column
-- ----------------------------------------------------------------------------

UPDATE opportunities o
SET job_type_id = jt.job_type_id
FROM job_types jt
WHERE jt.code = o.job::TEXT;


-- ----------------------------------------------------------------------------
-- 5. Now that all rows are populated, make job_type_id NOT NULL
-- ----------------------------------------------------------------------------

ALTER TABLE opportunities
    ALTER COLUMN job_type_id SET NOT NULL;


-- ----------------------------------------------------------------------------
-- 6. Drop the old CHECK constraint and enum column
-- ----------------------------------------------------------------------------

ALTER TABLE opportunities
    DROP CONSTRAINT IF EXISTS opportunities_job_check;

ALTER TABLE opportunities
    DROP COLUMN job;


-- ----------------------------------------------------------------------------
-- 7. Drop the old enum type
--    (safe now that no columns reference it)
-- ----------------------------------------------------------------------------

DROP TYPE job_type;
