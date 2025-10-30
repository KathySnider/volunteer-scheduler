-- ============================================
-- VOLUNTEER SCHEDULER DATABASE SCHEMA
-- Rollback Migration - Drops all tables and types
-- ============================================

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS event_attendees CASCADE;
DROP TABLE IF EXISTS volunteer_preferences CASCADE;
DROP TABLE IF EXISTS volunteer_shifts CASCADE;
DROP TABLE IF EXISTS shifts CASCADE;
DROP TABLE IF EXISTS opportunity_requirements CASCADE;
DROP TABLE IF EXISTS opportunities CASCADE;
DROP TABLE IF EXISTS staff CASCADE;
DROP TABLE IF EXISTS volunteer_qualifications CASCADE;
DROP TABLE IF EXISTS volunteers CASCADE;
DROP TABLE IF EXISTS event_dates CASCADE;
DROP TABLE IF EXISTS events CASCADE;
DROP TABLE IF EXISTS locations CASCADE;

-- Drop ENUM types
DROP TYPE IF EXISTS role_type;
DROP TYPE IF EXISTS qualification_type;