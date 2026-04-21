-- ============================================================================
-- VOLUNTEER SCHEDULER - FULL SCHEMA ROLLBACK
-- Drops everything created by 000001_init.up.sql in reverse dependency order.
-- ============================================================================

DROP TABLE IF EXISTS feedback_attachments;
DROP TABLE IF EXISTS feedback_notes;
DROP TABLE IF EXISTS feedback;

DROP TABLE IF EXISTS volunteer_shifts;
DROP TABLE IF EXISTS shifts;
DROP TABLE IF EXISTS opportunities;

DROP TABLE IF EXISTS event_service_types;
DROP TABLE IF EXISTS service_types;
DROP TABLE IF EXISTS event_dates;
DROP TABLE IF EXISTS events;

DROP TABLE IF EXISTS funding_entities;
DROP TABLE IF EXISTS venues;

DROP TABLE IF EXISTS job_types;

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS magic_links;

DROP TABLE IF EXISTS staff;
DROP TABLE IF EXISTS volunteers;

DROP TYPE IF EXISTS feedback_note_type;
DROP TYPE IF EXISTS feedback_status;
DROP TYPE IF EXISTS feedback_type;
DROP TYPE IF EXISTS volunteer_role;
