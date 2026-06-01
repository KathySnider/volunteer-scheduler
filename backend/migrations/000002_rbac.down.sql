-- ============================================================================
-- MIGRATION 000002: Additive RBAC — rollback
-- ============================================================================

-- 1. Re-create the enum.
CREATE TYPE volunteer_role AS ENUM ('VOLUNTEER', 'ADMINISTRATOR');

-- 2. Re-add the role columns with a temporary nullable state so we can
--    back-fill data before enforcing NOT NULL.
ALTER TABLE volunteers ADD COLUMN role volunteer_role;
ALTER TABLE sessions   ADD COLUMN role volunteer_role;

-- 3. Back-fill volunteers.role from volunteer_roles.
--    A volunteer with ADMINISTRATOR in volunteer_roles → ADMINISTRATOR;
--    otherwise → VOLUNTEER.
UPDATE volunteers v
SET    role = CASE
    WHEN EXISTS (
        SELECT 1
        FROM   volunteer_roles vr
        JOIN   roles r ON r.role_id = vr.role_id
        WHERE  vr.volunteer_id = v.volunteer_id
          AND  r.role_name = 'ADMINISTRATOR'
    ) THEN 'ADMINISTRATOR'::volunteer_role
    ELSE 'VOLUNTEER'::volunteer_role
END;

-- 4. Back-fill sessions.role from the volunteer's restored role.
UPDATE sessions s
SET    role = v.role
FROM   volunteers v
WHERE  s.volunteer_id = v.volunteer_id;

-- 5. Enforce NOT NULL now that columns are populated.
ALTER TABLE volunteers ALTER COLUMN role SET NOT NULL;
ALTER TABLE volunteers ALTER COLUMN role SET DEFAULT 'VOLUNTEER';
ALTER TABLE sessions   ALTER COLUMN role SET NOT NULL;
ALTER TABLE sessions   ALTER COLUMN role SET DEFAULT 'VOLUNTEER';

-- 6. Drop junction and lookup tables.
DROP TABLE volunteer_roles;
DROP TABLE roles;
