-- Rename enum values to align with updated domain language.
-- VOLUNTEER_REPLY → VOLUNTEER_NOTE (volunteers add context, not just replies)
-- RESOLVED_GITHUB → RESOLVED_IMPLEMENTED (not all resolutions involve GitHub)
-- Drop github_issue_url column (URLs go in notes instead).

ALTER TYPE feedback_note_type RENAME VALUE 'VOLUNTEER_REPLY' TO 'VOLUNTEER_NOTE';
ALTER TYPE feedback_status RENAME VALUE 'RESOLVED_GITHUB' TO 'RESOLVED_IMPLEMENTED';

ALTER TABLE feedback DROP COLUMN IF EXISTS github_issue_url;
