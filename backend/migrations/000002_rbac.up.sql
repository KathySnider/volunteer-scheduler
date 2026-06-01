-- ============================================================================
-- MIGRATION 000002: Additive RBAC
--
-- Replaces the single volunteer_role column with a roles table and a
-- volunteer_roles junction table so a volunteer can hold multiple roles.
-- Business rule: ADMINISTRATOR always implies VOLUNTEER (enforced in the
-- application layer, not at the DB level).
-- ============================================================================

-- 1. New lookup table.
CREATE TABLE roles (
    role_id   SERIAL PRIMARY KEY,
    role_name VARCHAR(50) NOT NULL UNIQUE
);

INSERT INTO roles (role_name) VALUES ('VOLUNTEER'), ('ADMINISTRATOR');

-- 2. Junction table.
CREATE TABLE volunteer_roles (
    volunteer_id INT NOT NULL REFERENCES volunteers(volunteer_id) ON DELETE CASCADE,
    role_id      INT NOT NULL REFERENCES roles(role_id)           ON DELETE CASCADE,
    PRIMARY KEY (volunteer_id, role_id)
);

-- 3. Migrate existing data.
--    Every volunteer gets the VOLUNTEER role.
INSERT INTO volunteer_roles (volunteer_id, role_id)
SELECT v.volunteer_id, r.role_id
FROM   volunteers v
CROSS  JOIN roles r
WHERE  r.role_name = 'VOLUNTEER';

--    Volunteers already marked ADMINISTRATOR also get the ADMINISTRATOR role.
INSERT INTO volunteer_roles (volunteer_id, role_id)
SELECT v.volunteer_id, r.role_id
FROM   volunteers v
JOIN   roles r ON r.role_name = 'ADMINISTRATOR'
WHERE  v.role = 'ADMINISTRATOR';

-- 4. Drop old columns and enum.
ALTER TABLE volunteers DROP COLUMN role;
ALTER TABLE sessions   DROP COLUMN role;
DROP TYPE volunteer_role;
