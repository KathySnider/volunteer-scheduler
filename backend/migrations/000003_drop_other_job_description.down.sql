-- ============================================================================
-- MIGRATION 000003 DOWN: Restore opportunities.other_job_description
-- ============================================================================

ALTER TABLE opportunities
    ADD COLUMN other_job_description TEXT;
