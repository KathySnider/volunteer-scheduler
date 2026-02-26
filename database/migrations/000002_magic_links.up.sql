-- Create magic_links table for passwordless email authentication
CREATE TABLE magic_links (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    ip_address VARCHAR(45),
    user_agent TEXT
);

-- Index for fast token lookups
CREATE INDEX idx_magic_links_token ON magic_links(token);
CREATE INDEX idx_magic_links_email ON magic_links(email, created_at DESC);

-- Optionally extend the volunteers table if needed for user management
-- (assuming you have a volunteers table already)
ALTER TABLE volunteers ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE volunteers ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMP;
