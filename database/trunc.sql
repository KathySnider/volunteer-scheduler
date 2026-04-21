-- ============================================================================
-- TRUNCATE ALL TRANSACTIONAL DATA
-- Resets sequences so IDs start from 1 again.
--
-- Does NOT truncate lookup tables seeded by the migration:
--   job_types, service_types, funding_entities
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
  venues,
  volunteers,
  staff,
  magic_links,
  sessions
RESTART IDENTITY CASCADE;
