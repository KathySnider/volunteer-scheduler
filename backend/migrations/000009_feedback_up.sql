-- Migration 000009: Add feedback table and feedback_added junction table.

CREATE TYPE FeedbackType AS ENUM (
    'BUG',
    'ENHANCEMENT',
    'GENERAL',
);

CREATE TYPE FeedbackStatus AS ENUM (
  'OPEN',
  'QUESTION_SENT', 
  'RESOLVED_GITHUB', 
  'RESOLVED_REJECTED'
);

CREATE TABLE feedback (
    feedback_id SERIAL PRIMARY KEY,
    volunteer_id INT NOT NULL REFERENCES volunteers (volunteer_id),
    feedback_type FeedbackType NOT NULL DEFAULT 'GENERAL'
    status FeedbackStatus NOT NULL DEFAULT 'OPEN',
    subject VARCHAR(50) NOT NULL
    app_page_name VARCHAR(50) NOT NULL,
    text TEXT NOT NULL,
    admin_notes TEXT,
    github_issue_url TEXT,
    created_at TIMESTAMP NOT NULL, 
    last_updated_at TIMESTAMP,
    resolved_at TIMESTAMP,
);

