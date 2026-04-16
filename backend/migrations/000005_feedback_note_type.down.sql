-- MIGRATION 000005 DOWN: Remove note_type from feedback_notes

DROP INDEX IF EXISTS idx_feedback_notes_note_type;

ALTER TABLE feedback_notes DROP COLUMN IF EXISTS note_type;

DROP TYPE IF EXISTS feedback_note_type;
