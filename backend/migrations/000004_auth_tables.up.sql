-- Migration 000004: Add passwordless auth tables and volunteer login fields.

-- Magic links for passwordless email authentication.
CREATE TABLE magic_links (
    id           SERIAL PRIMARY KEY,
    email        VARCHAR(255) NOT NULL,
    token        VARCHAR(255) UNIQUE NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at   TIMESTAMP NOT NULL,
    used_at      TIMESTAMP,
    ip_address   VARCHAR(45),
    user_agent   TEXT
);

CREATE INDEX idx_magic_links_token ON magic_links(token);
CREATE INDEX idx_magic_links_email ON magic_links(email, created_at DESC);

-- Sessions table for managing user sessions.
CREATE TABLE sessions (
    id               SERIAL PRIMARY KEY,
    email            VARCHAR(255) NOT NULL UNIQUE,
    token            VARCHAR(255) UNIQUE NOT NULL,
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at       TIMESTAMP NOT NULL,
    last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_token     ON sessions(token);
CREATE INDEX idx_sessions_email     ON sessions(email);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Add login tracking fields to volunteers.
ALTER TABLE volunteers
    ADD COLUMN IF NOT EXISTS created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMP;
