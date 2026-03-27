-- ============================================================================
-- MIGRATION 000002 DOWN: Restore job_type enum, revert opportunities table
-- ============================================================================


-- ----------------------------------------------------------------------------
-- 1. Recreate the original enum
-- ----------------------------------------------------------------------------

CREATE TYPE job_type AS ENUM (
    'event_support',
    'advocacy',
    'speaker',
    'volunteer_lead',
    'attendee_only',
    'other'
);


-- ----------------------------------------------------------------------------
-- 2. Add the enum column back (nullable during migration)
-- ----------------------------------------------------------------------------

ALTER TABLE opportunities
    ADD COLUMN job job_type;


-- ----------------------------------------------------------------------------
-- 3. Populate from job_types.code (any custom codes added after migration
--    will be set to 'other' as the closest safe fallback)
-- ----------------------------------------------------------------------------

UPDATE opportunities o
SET job = CASE jt.code
    WHEN 'event_support'  THEN 'event_support'::job_type
    WHEN 'advocacy'       THEN 'advocacy'::job_type
    WHEN 'speaker'        THEN 'speaker'::job_type
    WHEN 'volunteer_lead' THEN 'volunteer_lead'::job_type
    WHEN 'attendee_only'  THEN 'attendee_only'::job_type
    ELSE 'other'::job_type
END
FROM job_types jt
WHERE jt.job_type_id = o.job_type_id;


-- ----------------------------------------------------------------------------
-- 4. Restore NOT NULL and the original CHECK constraint
-- ----------------------------------------------------------------------------

ALTER TABLE opportunities
    ALTER COLUMN job SET NOT NULL;

ALTER TABLE opportunities
    ADD CONSTRAINT opportunities_job_check CHECK (
        (job != 'other' AND other_job_description IS NULL) OR
        (job  = 'other' AND other_job_description IS NOT NULL)
    );


-- ----------------------------------------------------------------------------
-- 5. Drop the FK column and the lookup table
-- ----------------------------------------------------------------------------

ALTER TABLE opportunities
    DROP COLUMN job_type_id;

DROP INDEX IF EXISTS idx_job_types_active;
DROP TABLE job_types;
