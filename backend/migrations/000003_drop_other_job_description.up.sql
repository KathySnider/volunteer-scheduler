-- ============================================================================
-- MIGRATION 000003: Drop opportunities.other_job_description
--
-- Rationale: other_job_description was a workaround for the rigid job_type
-- enum. Now that job types are admin-editable via the job_types table,
-- this field is no longer needed.
-- ============================================================================

ALTER TABLE opportunities
    DROP COLUMN other_job_description;
