-- ============================================
-- MIGRATION 000001 - Initial Schema ROLLBACK
-- ============================================

-- Drop indexes first
DROP INDEX IF EXISTS idx_volunteer_shifts_shift;
DROP INDEX IF EXISTS idx_volunteer_shifts_volunteer;
DROP INDEX IF EXISTS idx_shifts_staff_contact;
DROP INDEX IF EXISTS idx_shifts_opportunity;
DROP INDEX IF EXISTS idx_opportunities_event;
DROP INDEX IF EXISTS idx_volunteers_zip;
DROP INDEX IF EXISTS idx_volunteers_email;
DROP INDEX IF EXISTS idx_event_dates_date;
DROP INDEX IF EXISTS idx_event_dates_event;
DROP INDEX IF EXISTS idx_events_venue;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS event_service_types;
DROP TABLE IF EXISTS service_types;
DROP TABLE IF EXISTS volunteer_shifts;
DROP TABLE IF EXISTS shifts;
DROP TABLE IF EXISTS opportunities;
DROP TABLE IF EXISTS event_dates;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS volunteers;
DROP TABLE IF EXISTS staff;
DROP TABLE IF EXISTS venues;

-- Drop ENUM types
DROP TYPE IF EXISTS job_type;
