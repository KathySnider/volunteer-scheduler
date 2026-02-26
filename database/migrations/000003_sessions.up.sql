-- Create sessions table for managing user sessions
CREATE TABLE sessions (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    token VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for fast token lookups
CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_email ON sessions(email);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
