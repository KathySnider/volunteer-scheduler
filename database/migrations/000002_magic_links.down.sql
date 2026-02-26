-- Rollback magic_links table
DROP INDEX IF EXISTS idx_magic_links_email;
DROP INDEX IF EXISTS idx_magic_links_token;
DROP TABLE IF EXISTS magic_links;

-- Rollback added columns (optional - keep if you want to preserve data)
-- ALTER TABLE volunteers DROP COLUMN IF EXISTS created_at;
-- ALTER TABLE volunteers DROP COLUMN IF EXISTS last_login_at;
