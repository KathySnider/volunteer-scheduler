-- MIGRATION 000005: Add note_type to feedback_notes
--
-- Distinguishes who can see each note:
--   ADMIN_NOTE        -- internal; admins only, never shown to volunteer
--   QUESTION          -- admin asked the volunteer a question; visible to volunteer, triggers email
--   VOLUNTEER_REPLY   -- volunteer replied to a question; visible to both
--   EMAIL_TO_VOLUNTEER -- closing message on resolve/reject; visible to volunteer, triggers email
--
-- All pre-migration rows were written by admins via the original admin-only
-- mutations, so ADMIN_NOTE is the correct default for existing data.

CREATE TYPE feedback_note_type AS ENUM (
    'ADMIN_NOTE',
    'QUESTION',
    'VOLUNTEER_REPLY',
    'EMAIL_TO_VOLUNTEER'
);

ALTER TABLE feedback_notes
    ADD COLUMN note_type feedback_note_type NOT NULL DEFAULT 'ADMIN_NOTE';

CREATE INDEX idx_feedback_notes_note_type ON feedback_notes (note_type);
