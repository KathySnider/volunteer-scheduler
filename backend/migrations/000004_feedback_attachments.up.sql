-- Stores binary file attachments for feedback items.
-- file_data uses PostgreSQL's bytea type, which can hold up to 1 GB.
-- A 5 MB application-level limit is enforced in the service layer.
-- Multiple attachments per feedback item are supported.

CREATE TABLE feedback_attachments (
    attachment_id  SERIAL      PRIMARY KEY,
    feedback_id    INTEGER     NOT NULL REFERENCES feedback(feedback_id),
    filename       TEXT        NOT NULL,
    mime_type      TEXT        NOT NULL,
    file_data      BYTEA       NOT NULL,
    file_size      INTEGER     NOT NULL,   -- size in bytes, redundant but avoids re-scanning
    created_at     TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_feedback_attachments_feedback_id
    ON feedback_attachments (feedback_id);
