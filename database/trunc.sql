-- ============================================================================
-- TRUNCATE ALL TRANSACTIONAL DATA
-- Resets sequences so IDs start from 1 again.
--
-- Does NOT truncate lookup tables seeded by migrations:
--   job_types, service_types, regions
-- Those are re-created fresh on every `docker-compose down -v` + rebuild.
-- ============================================================================

TRUNCATE TABLE
  feedback_attachments,
  feedback_notes,
  feedback,
  volunteer_shifts,
  shifts,
  opportunities,
  event_service_types,
  event_dates,
  events,
  venue_regions,
  venues,
  volunteers,
  staff,
  magic_links,
  sessions
RESTART IDENTITY CASCADE;
